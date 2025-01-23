#!/usr/bin/env python3

import os
import json
import logging
from datetime import datetime, timedelta
from typing import List, Dict, Any, Tuple

import pandas as pd
import numpy as np
import yfinance as yf
from ta.trend import SMAIndicator, EMAIndicator, MACD
from ta.momentum import RSIIndicator
from ta.volatility import BollingerBands
from sklearn.preprocessing import MinMaxScaler
import requests

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class DataPreparer:
    def __init__(self):
        self.symbols = self._load_symbols()
        self.timeframes = ['1h', '4h', '24h', '7d']
        self.scaler = MinMaxScaler()
        
    def _load_symbols(self, file_path='config/symbols.json') -> List[str]:
        try:
            with open(file_path, 'r') as f:
                config = json.load(f)
                return config['symbols']
        except Exception as e:
            logger.error(f"Error loading symbols: {e}")
            return ['BTC-USD', 'ETH-USD']
    
    def fetch_historical_data(self, symbol: str, period: str = '2y') -> pd.DataFrame:
        try:
            ticker = yf.Ticker(symbol)
            df = ticker.history(period=period, interval='1h')
            logger.info(f"Fetched {len(df)} rows for {symbol}")
            return df
        except Exception as e:
            logger.error(f"Error fetching data for {symbol}: {e}")
            return None

    def calculate_technical_indicators(self, df: pd.DataFrame) -> pd.DataFrame:
        try:
            df['sma_20'] = SMAIndicator(df['Close'], window=20).sma_indicator()
            df['sma_50'] = SMAIndicator(df['Close'], window=50).sma_indicator()
            df['sma_200'] = SMAIndicator(df['Close'], window=200).sma_indicator()
            
            df['ema_12'] = EMAIndicator(df['Close'], window=12).ema_indicator()
            df['ema_26'] = EMAIndicator(df['Close'], window=26).ema_indicator()
            
            macd = MACD(df['Close'])
            df['macd'] = macd.macd()
            df['macd_signal'] = macd.macd_signal()
            df['macd_diff'] = macd.macd_diff()
            
            df['rsi'] = RSIIndicator(df['Close']).rsi()
            
            bb = BollingerBands(df['Close'])
            df['bb_high'] = bb.bollinger_hband()
            df['bb_low'] = bb.bollinger_lband()
            df['bb_mid'] = bb.bollinger_mavg()
            
            df['returns'] = df['Close'].pct_change()
            df['log_returns'] = np.log1p(df['returns'])
            df['volatility'] = df['returns'].rolling(window=20).std()
            
            df['volume_sma'] = df['Volume'].rolling(window=20).mean()
            df['volume_ratio'] = df['Volume'] / df['volume_sma']
            
            return df
        except Exception as e:
            logger.error(f"Error calculating indicators: {e}")
            return df

    def process_timeframes(self, df: pd.DataFrame) -> Dict[str, pd.DataFrame]:
        timeframe_data = {}
        
        for timeframe in self.timeframes:
            try:
                if timeframe == '1h':
                    resampled = df
                elif timeframe == '4h':
                    resampled = df.resample('4H').agg({
                        'Open': 'first',
                        'High': 'max',
                        'Low': 'min',
                        'Close': 'last',
                        'Volume': 'sum'
                    })
                elif timeframe == '24h':
                    resampled = df.resample('D').agg({
                        'Open': 'first',
                        'High': 'max',
                        'Low': 'min',
                        'Close': 'last',
                        'Volume': 'sum'
                    })
                else:  # 7d
                    resampled = df.resample('7D').agg({
                        'Open': 'first',
                        'High': 'max',
                        'Low': 'min',
                        'Close': 'last',
                        'Volume': 'sum'
                    })
                
                resampled = self.calculate_technical_indicators(resampled)
                
                resampled['target_high'] = resampled['High'].shift(-1)
                resampled['target_low'] = resampled['Low'].shift(-1)
                resampled['target_close'] = resampled['Close'].shift(-1)
                resampled['target_direction'] = (resampled['Close'].shift(-1) > resampled['Close']).astype(int)
                
                timeframe_data[timeframe] = resampled
                
            except Exception as e:
                logger.error(f"Error processing timeframe {timeframe}: {e}")
                continue
                
        return timeframe_data

    def prepare_features(self, df: pd.DataFrame) -> pd.DataFrame:
        feature_columns = [
            'Close', 'Volume', 'returns', 'log_returns', 'volatility',
            'sma_20', 'sma_50', 'sma_200', 'ema_12', 'ema_26',
            'macd', 'macd_signal', 'macd_diff', 'rsi',
            'bb_high', 'bb_low', 'bb_mid', 'volume_ratio'
        ]
        
        features = df[feature_columns].copy()
        features = features.fillna(method='ffill').fillna(method='bfill')
        
        features_scaled = pd.DataFrame(
            self.scaler.fit_transform(features),
            columns=features.columns,
            index=features.index
        )
        
        return features_scaled

    def split_sequences(self, data: pd.DataFrame, sequence_length: int = 60) -> Tuple[np.ndarray, Dict[str, np.ndarray]]:
        sequences = []
        targets = {
            'high': [],
            'low': [],
            'close': [],
            'direction': []
        }
        
        for i in range(len(data) - sequence_length):
            sequence = data.iloc[i:i + sequence_length]
            target_idx = i + sequence_length
            
            sequences.append(sequence.values)
            targets['high'].append(data.iloc[target_idx]['target_high'])
            targets['low'].append(data.iloc[target_idx]['target_low'])
            targets['close'].append(data.iloc[target_idx]['target_close'])
            targets['direction'].append(data.iloc[target_idx]['target_direction'])
        
        return np.array(sequences), {
            k: np.array(v) for k, v in targets.items()
        }

    def save_dataset(self, symbol: str, timeframe: str, sequences: np.ndarray, 
                    targets: Dict[str, np.ndarray], output_dir: str = 'data/processed'):
        try:
            if not os.path.exists(output_dir):
                os.makedirs(output_dir)

            dataset_path = os.path.join(output_dir, f"{symbol}_{timeframe}")
            if not os.path.exists(dataset_path):
                os.makedirs(dataset_path)

            np.save(os.path.join(dataset_path, 'sequences.npy'), sequences)
            for target_name, target_data in targets.items():
                np.save(os.path.join(dataset_path, f'target_{target_name}.npy'), target_data)

            # Save scaler
            with open(os.path.join(dataset_path, 'scaler.json'), 'w') as f:
                json.dump({
                    'scale_': self.scaler.scale_.tolist(),
                    'min_': self.scaler.min_.tolist(),
                    'data_min_': self.scaler.data_min_.tolist(),
                    'data_max_': self.scaler.data_max_.tolist(),
                    'data_range_': self.scaler.data_range_.tolist()
                }, f)

            # Save metadata
            metadata = {
                'symbol': symbol,
                'timeframe': timeframe,
                'sequence_length': sequences.shape[1],
                'feature_dim': sequences.shape[2],
                'num_sequences': sequences.shape[0],
                'created_at': datetime.now().isoformat(),
                'feature_names': self.get_feature_names()
            }
            with open(os.path.join(dataset_path, 'metadata.json'), 'w') as f:
                json.dump(metadata, f, indent=2)

            logger.info(f"Saved dataset for {symbol} {timeframe}")
        except Exception as e:
            logger.error(f"Error saving dataset: {e}")

    def get_feature_names(self) -> List[str]:
        return [
            'Close', 'Volume', 'returns', 'log_returns', 'volatility',
            'sma_20', 'sma_50', 'sma_200', 'ema_12', 'ema_26',
            'macd', 'macd_signal', 'macd_diff', 'rsi',
            'bb_high', 'bb_low', 'bb_mid', 'volume_ratio'
        ]

    def prepare_all_data(self):
        for symbol in self.symbols:
            logger.info(f"Processing {symbol}")
            
            # Fetch data
            df = self.fetch_historical_data(symbol)
            if df is None:
                continue
                
            # Process each timeframe
            timeframe_data = self.process_timeframes(df)
            
            for timeframe, data in timeframe_data.items():
                logger.info(f"Preparing {timeframe} data for {symbol}")
                
                # Prepare features
                features = self.prepare_features(data)
                
                # Create sequences
                sequences, targets = self.split_sequences(features)
                
                # Save dataset
                self.save_dataset(symbol, timeframe, sequences, targets)

def main():
    preparer = DataPreparer()
    preparer.prepare_all_data()
    logger.info("Data preparation completed")

if __name__ == "__main__":
    main()