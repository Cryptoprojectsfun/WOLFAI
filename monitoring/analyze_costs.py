#!/usr/bin/env python3

import os
import json
import logging
from datetime import datetime, timedelta

import requests
import pandas as pd
import numpy as np

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class CostAnalyzer:
    def __init__(self):
        self.cost_threshold = float(os.getenv("COST_THRESHOLD", "1000"))
        self.slack_webhook_url = os.getenv("SLACK_WEBHOOK_URL")
        self.alert_threshold_percent = 0.8  # Alert at 80% of threshold

    def load_cost_data(self, file_path='cost_data.json'):
        """Load cost data from AWS Cost Explorer output"""
        try:
            with open(file_path, 'r') as f:
                data = json.load(f)
            
            cost_data = []
            for result in data['ResultsByTime']:
                date = result['TimePeriod']['Start']
                for group in result['Groups']:
                    cost_data.append({
                        'date': date,
                        'service': group['Keys'][0],
                        'cost': float(group['Metrics']['BlendedCost']['Amount']),
                        'unit': group['Metrics']['BlendedCost']['Unit']
                    })
            
            return pd.DataFrame(cost_data)
        except Exception as e:
            logger.error(f"Error loading cost data: {e}")
            return None

    def analyze_daily_costs(self, df):
        """Analyze daily cost patterns"""
        daily_costs = df.groupby('date')['cost'].sum().reset_index()
        daily_costs['date'] = pd.to_datetime(daily_costs['date'])
        
        stats = {
            'total_cost': daily_costs['cost'].sum(),
            'avg_daily_cost': daily_costs['cost'].mean(),
            'max_daily_cost': daily_costs['cost'].max(),
            'cost_trend': daily_costs['cost'].pct_change().mean(),
            'projected_monthly_cost': daily_costs['cost'].mean() * 30
        }
        
        return stats

    def analyze_service_costs(self, df):
        """Analyze costs by service"""
        service_costs = df.groupby('service').agg({
            'cost': ['sum', 'mean', 'std']
        }).reset_index()
        
        service_costs.columns = ['service', 'total_cost', 'avg_cost', 'std_cost']
        service_costs['cost_percentage'] = (
            service_costs['total_cost'] / service_costs['total_cost'].sum() * 100
        )
        
        return service_costs

    def analyze_anomalies(self, df):
        """Detect cost anomalies"""
        daily_costs = df.groupby('date')['cost'].sum().reset_index()
        
        # Calculate Z-scores
        daily_costs['zscore'] = (
            (daily_costs['cost'] - daily_costs['cost'].mean()) / 
            daily_costs['cost'].std()
        )
        
        # Identify anomalies (Z-score > 2)
        anomalies = daily_costs[abs(daily_costs['zscore']) > 2]
        
        return anomalies if not anomalies.empty else None

    def check_thresholds(self, daily_stats):
        """Check if costs are approaching threshold"""
        projected_cost = daily_stats['projected_monthly_cost']
        
        alerts = []
        if projected_cost >= self.cost_threshold:
            alerts.append({
                'level': 'CRITICAL',
                'message': f'Projected monthly cost (${projected_cost:.2f}) '
                          f'exceeds threshold (${self.cost_threshold:.2f})'
            })
        elif projected_cost >= self.cost_threshold * self.alert_threshold_percent:
            alerts.append({
                'level': 'WARNING',
                'message': f'Projected monthly cost (${projected_cost:.2f}) '
                          f'approaching threshold (${self.cost_threshold:.2f})'
            })
            
        return alerts

    def send_alert(self, alerts, daily_stats, service_costs, anomalies=None):
        """Send alerts to Slack"""
        if not alerts:
            return

        message = {
            "blocks": [
                {
                    "type": "header",
                    "text": {
                        "type": "plain_text",
                        "text": "⚠️ Cost Alert"
                    }
                },
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": alerts[0]['message']
                    }
                },
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": f"*Daily Cost Summary*\n"
                               f"• Average: ${daily_stats['avg_daily_cost']:.2f}\n"
                               f"• Maximum: ${daily_stats['max_daily_cost']:.2f}\n"
                               f"• Trend: {daily_stats['cost_trend']*100:.1f}%"
                    }
                }
            ]
        }

        # Add top services by cost
        top_services = service_costs.nlargest(3, 'total_cost')
        services_text = "*Top Services by Cost*\n" + "\n".join(
            f"• {row['service']}: ${row['total_cost']:.2f} "
            f"({row['cost_percentage']:.1f}%)"
            for _, row in top_services.iterrows()
        )
        message["blocks"].append({
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": services_text
            }
        })

        # Add anomalies if any
        if anomalies is not None and not anomalies.empty:
            anomalies_text = "*Cost Anomalies Detected*\n" + "\n".join(
                f"• {row['date'].strftime('%Y-%m-%d')}: ${row['cost']:.2f} "
                f"(z-score: {row['zscore']:.1f})"
                for _, row in anomalies.iterrows()
            )
            message["blocks"].append({
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": anomalies_text
                }
            })

        try:
            response = requests.post(
                self.slack_webhook_url,
                json=message
            )
            response.raise_for_status()
        except Exception as e:
            logger.error(f"Error sending Slack alert: {e}")

def main():
    analyzer = CostAnalyzer()
    
    # Load and analyze cost data
    df = analyzer.load_cost_data()
    if df is None:
        return

    daily_stats = analyzer.analyze_daily_costs(df)
    service_costs = analyzer.analyze_service_costs(df)
    anomalies = analyzer.analyze_anomalies(df)
    
    # Check for alerts
    alerts = analyzer.check_thresholds(daily_stats)
    if alerts:
        analyzer.send_alert(alerts, daily_stats, service_costs, anomalies)
    
    # Export metrics for Prometheus
    export_metrics(daily_stats, service_costs)

def export_metrics(daily_stats, service_costs):
    """Export cost metrics in Prometheus format"""
    metrics = []
    
    # Daily cost metrics
    for metric, value in daily_stats.items():
        metrics.append(f"aws_cost_{metric} {value}")
    
    # Service cost metrics
    for _, row in service_costs.iterrows():
        service_name = row['service'].replace('-', '_').replace(':', '_')
        metrics.append(
            f'aws_cost_service_total{{service="{service_name}"}} {row["total_cost"]}'
        )
        metrics.append(
            f'aws_cost_service_percentage{{service="{service_name}"}} '
            f'{row["cost_percentage"]}'
        )
    
    # Write metrics file
    with open('cost_metrics.txt', 'w') as f:
        f.write('\n'.join(metrics))

if __name__ == "__main__":
    main()