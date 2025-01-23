#!/usr/bin/env python3

import os
import json
import logging
from datetime import datetime, timedelta

import requests
import pandas as pd
import numpy as np

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ModelPerformanceChecker:
    def __init__(self):
        self.alert_threshold = float(os.getenv("ALERT_THRESHOLD", "0.80"))
        self.slack_webhook_url = os.getenv("SLACK_WEBHOOK_URL")
        self.thresholds = {
            'directional_accuracy': 0.80,
            'interval_accuracy': 0.75,
            'rmse': 0.02,
            'profit_correlation': 0.60,
            'r2': 0.70,
            'avg_confidence': 0.70
        }

    def load_results(self, file_path='evaluation_results.json'):
        try:
            with open(file_path, 'r') as f:
                results = json.load(f)
            return results
        except Exception as e:
            logger.error(f"Error loading evaluation results: {e}")
            return None

    def check_performance(self, results):
        alerts = []
        summary = []

        for model_id, metrics in results['results'].items():
            model_alerts = []
            
            for metric, threshold in self.thresholds.items():
                if metric not in metrics:
                    continue
                    
                value = metrics[metric]
                if metric == 'rmse':
                    if value > threshold:
                        model_alerts.append({
                            'metric': metric,
                            'value': value,
                            'threshold': threshold,
                            'status': 'high'
                        })
                else:
                    if value < threshold:
                        model_alerts.append({
                            'metric': metric,
                            'value': value,
                            'threshold': threshold,
                            'status': 'low'
                        })

            if model_alerts:
                alerts.append({
                    'model_id': model_id,
                    'alerts': model_alerts,
                    'evaluation_time': metrics.get('evaluated_at')
                })

            summary.append({
                'model_id': model_id,
                'directional_accuracy': metrics.get('directional_accuracy', 0),
                'interval_accuracy': metrics.get('interval_accuracy', 0),
                'rmse': metrics.get('rmse', float('inf')),
                'profit_correlation': metrics.get('profit_correlation', 0),
                'alerts': len(model_alerts)
            })

        return alerts, summary

    def format_alert_message(self, alerts, summary):
        blocks = [
            {
                "type": "header",
                "text": {
                    "type": "plain_text",
                    "text": "🔍 Model Performance Report"
                }
            }
        ]

        if alerts:
            alert_text = "*Performance Alerts:*\n"
            for alert in alerts:
                alert_text += f"\n*Model {alert['model_id']}*\n"
                for issue in alert['alerts']:
                    status = "❌ Too high" if issue['status'] == 'high' else "❌ Too low"
                    alert_text += (
                        f"• {issue['metric']}: {issue['value']:.2%} "
                        f"({status}, threshold: {issue['threshold']:.2%})\n"
                    )
            
            blocks.append({
                "type": "section",
                "text": {
                    "type": "mrkdwn",
                    "text": alert_text
                }
            })

        summary_df = pd.DataFrame(summary)
        top_models = summary_df.nlargest(3, 'directional_accuracy')
        
        summary_text = "*Top Performing Models:*\n"
        for _, model in top_models.iterrows():
            summary_text += (
                f"\n*Model {model['model_id']}*\n"
                f"• Directional Accuracy: {model['directional_accuracy']:.2%}\n"
                f"• Interval Accuracy: {model['interval_accuracy']:.2%}\n"
                f"• RMSE: {model['rmse']:.4f}\n"
                f"• Profit Correlation: {model['profit_correlation']:.2f}\n"
            )

        blocks.append({
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": summary_text
            }
        })

        stats_text = "*Overall Statistics:*\n"
        stats = {
            "Average Directional Accuracy": summary_df['directional_accuracy'].mean(),
            "Best RMSE": summary_df['rmse'].min(),
            "Models with Alerts": summary_df['alerts'].sum(),
            "Total Models": len(summary_df)
        }

        for stat, value in stats.items():
            if isinstance(value, float):
                stats_text += f"• {stat}: {value:.2%}\n"
            else:
                stats_text += f"• {stat}: {value}\n"

        blocks.append({
            "type": "section",
            "text": {
                "type": "mrkdwn",
                "text": stats_text
            }
        })

        return {"blocks": blocks}

    def send_alert(self, alerts, summary):
        if not self.slack_webhook_url:
            return

        message = self.format_alert_message(alerts, summary)
        try:
            response = requests.post(
                self.slack_webhook_url,
                json=message
            )
            response.raise_for_status()
            logger.info("Alert sent successfully")
        except Exception as e:
            logger.error(f"Error sending alert: {e}")

    def export_metrics(self, summary):
        metrics = []
        summary_df = pd.DataFrame(summary)

        # Overall metrics
        metrics.extend([
            f'model_performance_directional_accuracy_avg {summary_df["directional_accuracy"].mean()}',
            f'model_performance_rmse_min {summary_df["rmse"].min()}',
            f'model_performance_alerts_total {summary_df["alerts"].sum()}'
        ])

        # Per-model metrics
        for _, row in summary_df.iterrows():
            model_id = row['model_id']
            base = f'model_performance{{model_id="{model_id}"}}'
            metrics.extend([
                f'{base}_directional_accuracy {row["directional_accuracy"]}',
                f'{base}_interval_accuracy {row["interval_accuracy"]}',
                f'{base}_rmse {row["rmse"]}',
                f'{base}_profit_correlation {row["profit_correlation"]}',
                f'{base}_alerts {row["alerts"]}'
            ])

        with open('model_performance_metrics.txt', 'w') as f:
            f.write('\n'.join(metrics))

def main():
    checker = ModelPerformanceChecker()
    results = checker.load_results()
    if results:
        alerts, summary = checker.check_performance(results)
        checker.send_alert(alerts, summary)
        checker.export_metrics(summary)
        logger.info("Performance check completed")
    else:
        logger.error("Performance check failed")

if __name__ == "__main__":
    main()