#!/usr/bin/env python3

import os
import json
import time
import logging
from datetime import datetime, timedelta

import requests
import pandas as pd
import numpy as np
from prometheus_client import CollectorRegistry, Gauge, write_text_file

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Constants
ENVIRONMENT = os.getenv("ENVIRONMENT", "production")
API_TOKEN = os.getenv("API_TOKEN")
API_BASE_URL = f"https://{'staging.' if ENVIRONMENT == 'staging' else ''}api.quantai.com"

class MetricsCollector:
    def __init__(self):
        self.registry = CollectorRegistry()
        self._initialize_metrics()

    def _initialize_metrics(self):
        # System metrics
        self.system_memory = Gauge(
            'system_memory_usage_bytes',
            'Memory usage in bytes',
            ['type'],
            registry=self.registry
        )
        
        self.system_cpu = Gauge(
            'system_cpu_usage_percent',
            'CPU usage percentage',
            ['type'],
            registry=self.registry
        )

        # Model metrics
        self.model_prediction_accuracy = Gauge(
            'model_prediction_accuracy',
            'Model prediction accuracy',
            ['model_id'],
            registry=self.registry
        )
        
        self.model_prediction_latency = Gauge(
            'model_prediction_latency_seconds',
            'Model prediction latency in seconds',
            ['model_id'],
            registry=self.registry
        )

        # Portfolio metrics
        self.portfolio_value = Gauge(
            'portfolio_total_value',
            'Total portfolio value',
            ['portfolio_id'],
            registry=self.registry
        )
        
        self.portfolio_return = Gauge(
            'portfolio_return_rate',
            'Portfolio return rate',
            ['portfolio_id', 'timeframe'],
            registry=self.registry
        )

        # API metrics
        self.api_request_count = Gauge(
            'api_request_total',
            'Total API requests',
            ['endpoint', 'method', 'status'],
            registry=self.registry
        )
        
        self.api_latency = Gauge(
            'api_request_latency_seconds',
            'API request latency in seconds',
            ['endpoint', 'method'],
            registry=self.registry
        )

    def collect_system_metrics(self):
        """Collect system-level metrics"""
        try:
            response = requests.get(
                f"{API_BASE_URL}/metrics/system",
                headers={"Authorization": f"Bearer {API_TOKEN}"}
            )
            response.raise_for_status()
            metrics = response.json()

            # Memory metrics
            self.system_memory.labels('heap').set(metrics['memory']['heap'])
            self.system_memory.labels('stack').set(metrics['memory']['stack'])
            self.system_memory.labels('total').set(metrics['memory']['total'])

            # CPU metrics
            self.system_cpu.labels('user').set(metrics['cpu']['user'])
            self.system_cpu.labels('system').set(metrics['cpu']['system'])
            self.system_cpu.labels('total').set(metrics['cpu']['total'])

        except Exception as e:
            logger.error(f"Error collecting system metrics: {e}")

    def collect_model_metrics(self):
        """Collect AI model performance metrics"""
        try:
            response = requests.get(
                f"{API_BASE_URL}/metrics/models",
                headers={"Authorization": f"Bearer {API_TOKEN}"}
            )
            response.raise_for_status()
            metrics = response.json()

            for model_id, model_metrics in metrics.items():
                self.model_prediction_accuracy.labels(
                    model_id=model_id
                ).set(model_metrics['accuracy'])
                
                self.model_prediction_latency.labels(
                    model_id=model_id
                ).set(model_metrics['avg_latency'])

        except Exception as e:
            logger.error(f"Error collecting model metrics: {e}")

    def collect_portfolio_metrics(self):
        """Collect portfolio performance metrics"""
        try:
            response = requests.get(
                f"{API_BASE_URL}/metrics/portfolios",
                headers={"Authorization": f"Bearer {API_TOKEN}"}
            )
            response.raise_for_status()
            metrics = response.json()

            for portfolio in metrics:
                pid = portfolio['portfolio_id']
                self.portfolio_value.labels(portfolio_id=pid).set(
                    portfolio['total_value']
                )
                
                for timeframe in ['daily', 'weekly', 'monthly', 'yearly']:
                    self.portfolio_return.labels(
                        portfolio_id=pid,
                        timeframe=timeframe
                    ).set(portfolio[f'{timeframe}_return'])

        except Exception as e:
            logger.error(f"Error collecting portfolio metrics: {e}")

    def collect_api_metrics(self):
        """Collect API performance metrics"""
        try:
            response = requests.get(
                f"{API_BASE_URL}/metrics/api",
                headers={"Authorization": f"Bearer {API_TOKEN}"}
            )
            response.raise_for_status()
            metrics = response.json()

            for endpoint, data in metrics['requests'].items():
                for method, counts in data['methods'].items():
                    for status, count in counts['status_codes'].items():
                        self.api_request_count.labels(
                            endpoint=endpoint,
                            method=method,
                            status=status
                        ).set(count)

                    self.api_latency.labels(
                        endpoint=endpoint,
                        method=method
                    ).set(data['latency']['avg'])

        except Exception as e:
            logger.error(f"Error collecting API metrics: {e}")

    def collect_all(self):
        """Collect all metrics"""
        logger.info("Starting metrics collection...")
        
        self.collect_system_metrics()
        self.collect_model_metrics()
        self.collect_portfolio_metrics()
        self.collect_api_metrics()
        
        logger.info("Metrics collection completed")

    def export_metrics(self, output_file='metrics.txt'):
        """Export collected metrics to a file"""
        write_text_file(self.registry, output_file)
        logger.info(f"Metrics exported to {output_file}")

def main():
    collector = MetricsCollector()
    collector.collect_all()
    collector.export_metrics()

if __name__ == "__main__":
    main()