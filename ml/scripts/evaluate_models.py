#!/usr/bin/env python3

import os
import json
import logging
from datetime import datetime, timedelta

import pandas as pd
import numpy as np
from sklearn.metrics import (
    accuracy_score, 
    precision_score, 
    recall_score, 
    f1_score,
    mean_squared_error,
    mean_absolute_error
)
import requests

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class ModelEvaluator:
    def __init__(self):
        self.api_url = os.getenv("MODEL_API_URL")
        self.api_token = os.getenv("MODEL_API_TOKEN")
        self.evaluation_data = None
        self.predictions = {}
        self.results = {}
        self.alert_threshold = float(os.getenv("ALERT_THRESHOLD", "0.8"))
        self.slack_webhook_url = os.getenv("SLACK_WEBHOOK_URL")

    def load_data(self, file_path='data/evaluation_data.csv'):
        """Load evaluation dataset"""
        try:
            self.evaluation_data = pd.read_csv(file_path)
            logger.info(f"Loaded {len(self.evaluation_data)} evaluation samples")
        except Exception as e:
            logger.error(f"Error loading evaluation data: {e}")
            return False
        return True

    def get_predictions(self, model_id, data):
        """Get predictions from model API"""
        predictions = []
        errors = 0
        
        headers = {
            "Authorization": f"Bearer {self.api_token}",
            "Content-Type": "application/json"
        }

        for _, row in data.iterrows():
            try:
                response = requests.post(
                    f"{self.api_url}/predict/{model_id}",
                    headers=headers,
                    json={
                        "features": row.to_dict(),
                        "timeframe": "24h"
                    }
                )
                response.raise_for_status()
                prediction = response.json()
                predictions.append({
                    'predicted_high': prediction['predicted_high'],
                    'predicted_low': prediction['predicted_low'],
                    'confidence': prediction['confidence']
                })
            except Exception as e:
                logger.error(f"Error getting prediction: {e}")
                errors += 1
                predictions.append(None)

        if errors > len(data) * 0.1:  # More than 10% errors
            logger.error(f"High error rate for model {model_id}: {errors} errors")
            return None

        return predictions

    def evaluate_classification_model(self, y_true, y_pred, confidence):
        """Evaluate classification model performance"""
        metrics = {
            'accuracy': accuracy_score(y_true, y_pred),
            'precision': precision_score(y_true, y_pred, average='weighted'),
            'recall': recall_score(y_true, y_pred, average='weighted'),
            'f1': f1_score(y_true, y_pred, average='weighted'),
            'avg_confidence': np.mean(confidence),
            'confidence_accuracy_correlation': np.corrcoef(
                confidence, 
                y_true == y_pred
            )[0,1]
        }
        return metrics

    def evaluate_regression_model(self, y_true, y_pred, confidence):
        """Evaluate regression model performance"""
        metrics = {
            'mse': mean_squared_error(y_true, y_pred),
            'rmse': np.sqrt(mean_squared_error(y_true, y_pred)),
            'mae': mean_absolute_error(y_true, y_pred),
            'r2': 1 - (
                np.sum((y_true - y_pred) ** 2) / 
                np.sum((y_true - np.mean(y_true)) ** 2)
            ),
            'avg_confidence': np.mean(confidence),
            'confidence_error_correlation': np.corrcoef(
                confidence, 
                np.abs(y_true - y_pred)
            )[0,1]
        }
        return metrics

    def evaluate_model(self, model_id):
        """Evaluate a single model"""
        predictions = self.get_predictions(model_id, self.evaluation_data)
        if predictions is None:
            return None

        y_true = self.evaluation_data['actual_price'].values
        y_pred = np.array([p['predicted_high'] for p in predictions])
        confidence = np.array([p['confidence'] for p in predictions])

        # Calculate prediction error metrics
        metrics = self.evaluate_regression_model(y_true, y_pred, confidence)

        # Calculate directional accuracy
        direction_true = np.diff(y_true) > 0
        direction_pred = np.diff(y_pred) > 0
        metrics['directional_accuracy'] = np.mean(
            direction_true == direction_pred
        )

        # Calculate prediction intervals accuracy
        in_interval = np.logical_and(
            y_true >= [p['predicted_low'] for p in predictions],
            y_true <= [p['predicted_high'] for p in predictions]
        )
        metrics['interval_accuracy'] = np.mean(in_interval)

        # Calculate profit potential
        returns = np.diff(y_true) / y_true[:-1]
        pred_returns = np.diff(y_pred) / y_pred[:-1]
        metrics['profit_correlation'] = np.corrcoef(
            returns, 
            pred_returns
        )[0,1]

        # Add timestamp
        metrics['evaluated_at'] = datetime.now().isoformat()

        return metrics

    def evaluate_all_models(self):
        """Evaluate all models"""
        if not self.load_data():
            return False

        # Get list of models from API
        try:
            response = requests.get(
                f"{self.api_url}/models",
                headers={"Authorization": f"Bearer {self.api_token}"}
            )
            response.raise_for_status()
            models = response.json()
        except Exception as e:
            logger.error(f"Error getting model list: {e}")
            return False

        # Evaluate each model
        for model in models:
            logger.info(f"Evaluating model {model['id']}")
            metrics = self.evaluate_model(model['id'])
            if metrics is not None:
                self.results[model['id']] = metrics
                
                # Check performance thresholds
                if metrics['directional_accuracy'] < self.alert_threshold:
                    self.send_alert(model['id'], metrics)

        # Save results
        self.save_results()
        return True

    def save_results(self, file_path='evaluation_results.json'):
        """Save evaluation results"""
        try:
            with open(file_path, 'w') as f:
                json.dump({
                    'timestamp': datetime.now().isoformat(),
                    'results': self.results
                }, f, indent=2)
            logger.info(f"Results saved to {file_path}")
            
            # Export Prometheus metrics
            self.export_metrics()
        except Exception as e:
            logger.error(f"Error saving results: {e}")

    def export_metrics(self):
        """Export metrics in Prometheus format"""
        metrics = []
        
        for model_id, result in self.results.items():
            for metric, value in result.items():
                if isinstance(value, (int, float)):
                    metric_name = f"model_{metric}"
                    metric_name = metric_name.replace('-', '_')
                    metrics.append(
                        f'{metric_name}{{model_id="{model_id}"}} {value}'
                    )

        with open('model_metrics.txt', 'w') as f:
            f.write('\n'.join(metrics))

    def send_alert(self, model_id, metrics):
        """Send alert for poor model performance"""
        if not self.slack_webhook_url:
            return

        message = {
            "blocks": [
                {
                    "type": "header",
                    "text": {
                        "type": "plain_text",
                        "text": "🚨 Model Performance Alert"
                    }
                },
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": f"*Model {model_id}* performance below threshold\n"
                               f"• Directional Accuracy: "
                               f"{metrics['directional_accuracy']:.2%}\n"
                               f"• RMSE: {metrics['rmse']:.4f}\n"
                               f"• Average Confidence: "
                               f"{metrics['avg_confidence']:.2%}"
                    }
                },
                {
                    "type": "section",
                    "text": {
                        "type": "mrkdwn",
                        "text": "Additional Metrics:\n"
                               f"• Interval Accuracy: "
                               f"{metrics['interval_accuracy']:.2%}\n"
                               f"• Profit Correlation: "
                               f"{metrics['profit_correlation']:.2f}\n"
                               f"• R²: {metrics['r2']:.4f}"
                    }
                }
            ]
        }

        try:
            response = requests.post(
                self.slack_webhook_url,
                json=message
            )
            response.raise_for_status()
        except Exception as e:
            logger.error(f"Error sending alert: {e}")

def main():
    evaluator = ModelEvaluator()
    if evaluator.evaluate_all_models():
        logger.info("Evaluation completed successfully")
    else:
        logger.error("Evaluation failed")

if __name__ == "__main__":
    main()