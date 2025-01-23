#!/usr/bin/env python3

import os
import json
import argparse
import logging
from typing import Tuple

import numpy as np
import pandas as pd
from sklearn.model_selection import train_test_split

from models.lstm import PricePredictor

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

def load_data(file_path: str) -> pd.DataFrame:
    """Load and prepare the training data"""
    df = pd.read_csv(file_path)
    
    # Add technical indicators
    df['SMA_20'] = df['close'].rolling(window=20).mean()
    df['SMA_50'] = df['close'].rolling(window=50).mean()
    df['RSI'] = calculate_rsi(df['close'])
    df['MACD'] = calculate_macd(df['close'])
    
    # Create target variables
    df['direction'] = (df['close'].shift(-1) > df['close']).astype(int)
    
    # Drop rows with NaN values
    df.dropna(inplace=True)
    return df

def calculate_rsi(prices: pd.Series, periods: int = 14) -> pd.Series:
    """Calculate Relative Strength Index"""
    delta = prices.diff()
    gain = (delta.where(delta > 0, 0)).rolling(window=periods).mean()
    loss = (-delta.where(delta < 0, 0)).rolling(window=periods).mean()
    
    rs = gain / loss
    return 100 - (100 / (1 + rs))

def calculate_macd(prices: pd.Series) -> pd.Series:
    """Calculate MACD indicator"""
    exp1 = prices.ewm(span=12, adjust=False).mean()
    exp2 = prices.ewm(span=26, adjust=False).mean()
    return exp1 - exp2

def prepare_features(df: pd.DataFrame) -> Tuple[np.ndarray, np.ndarray]:
    """Prepare feature matrix and target variables"""
    feature_columns = [
        'open', 'high', 'low', 'close', 'volume',
        'SMA_20', 'SMA_50', 'RSI', 'MACD'
    ]
    
    target_columns = ['high', 'low', 'close', 'direction']
    
    X = df[feature_columns].values
    y = df[target_columns].values
    
    return X, y

def main():
    parser = argparse.ArgumentParser(description='Train LSTM model for price prediction')
    parser.add_argument('--data', type=str, required=True, help='Path to training data CSV')
    parser.add_argument('--config', type=str, required=True, help='Path to model config JSON')
    parser.add_argument('--output', type=str, required=True, help='Output directory for model')
    args = parser.parse_args()
    
    # Load and prepare data
    logger.info("Loading and preparing data...")
    df = load_data(args.data)
    X, y = prepare_features(df)
    
    # Load model configuration
    with open(args.config) as f:
        config = json.load(f)
    
    # Create model instance
    model = PricePredictor(config)
    
    # Preprocess data
    X_scaled, _ = model.preprocess_data(X)
    X_seq, y_seq = model.create_sequences(X_scaled)
    
    # Split data
    X_train, X_val, y_train, y_val = train_test_split(
        X_seq, y_seq, test_size=0.2, shuffle=False
    )
    
    # Train model
    logger.info("Training model...")
    model.train(
        X_train, y_train,
        X_val, y_val,
        batch_size=config.get('batch_size', 32),
        epochs=config.get('epochs', 100)
    )
    
    # Save model
    logger.info(f"Saving model to {args.output}")
    os.makedirs(args.output, exist_ok=True)
    model.save(args.output)

if __name__ == '__main__':
    main()
