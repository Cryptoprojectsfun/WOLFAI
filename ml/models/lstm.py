#!/usr/bin/env python3

import os
import json
import logging
from typing import Dict, Tuple, List, Any

import numpy as np
import tensorflow as tf
from tensorflow.keras import layers, models, callbacks
from sklearn.preprocessing import MinMaxScaler

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

class PricePredictor:
    def __init__(self, config: Dict[str, Any]):
        self.config = config
        self.sequence_length = config.get('sequence_length', 60)
        self.feature_dim = config.get('feature_dim', 18)
        self.lstm_units = config.get('lstm_units', [128, 64])
        self.dropout_rate = config.get('dropout_rate', 0.2)
        self.learning_rate = config.get('learning_rate', 0.001)
        self.model = None
        self.scaler = None
        self.history = None
        self.callbacks = self._create_callbacks()

    def _create_model(self) -> models.Model:
        """Create the LSTM model architecture"""
        inputs = layers.Input(shape=(self.sequence_length, self.feature_dim))
        
        # LSTM layers with residual connections
        x = inputs
        for i, units in enumerate(self.lstm_units):
            lstm_out = layers.LSTM(
                units,
                return_sequences=(i < len(self.lstm_units) - 1),
                dropout=self.dropout_rate
            )(x)
            
            if i < len(self.lstm_units) - 1:
                # Add residual connection if dimensions match
                if units == self.feature_dim:
                    x = layers.Add()([x, lstm_out])
                x = lstm_out
            else:
                x = lstm_out

        # Attention mechanism
        attention = layers.Dense(1, activation='tanh')(x)
        attention = layers.Flatten()(attention)
        attention = layers.Activation('softmax')(attention)
        attention = layers.RepeatVector(self.lstm_units[-1])(attention)
        attention = layers.Permute([2, 1])(attention)
        
        # Apply attention to LSTM output
        x = layers.Multiply()([x, attention])
        
        # Dense layers for different prediction targets
        dense = layers.Dense(32, activation='relu')(x)
        dense = layers.BatchNormalization()(dense)
        dense = layers.Dropout(self.dropout_rate)(dense)
        
        # Multiple output heads
        price_high = layers.Dense(1, name='price_high')(dense)
        price_low = layers.Dense(1, name='price_low')(dense)
        price_close = layers.Dense(1, name='price_close')(dense)
        direction = layers.Dense(1, activation='sigmoid', name='direction')(dense)
        
        model = models.Model(
            inputs=inputs,
            outputs=[price_high, price_low, price_close, direction]
        )
        
        # Compile model with appropriate losses and metrics
        model.compile(
            optimizer=tf.keras.optimizers.Adam(learning_rate=self.learning_rate),
            loss={
                'price_high': 'mse',
                'price_low': 'mse',
                'price_close': 'mse',
                'direction': 'binary_crossentropy'
            },
            loss_weights={
                'price_high': 1.0,
                'price_low': 1.0,
                'price_close': 1.0,
                'direction': 0.5
            },
            metrics={
                'price_high': ['mae', 'mape'],
                'price_low': ['mae', 'mape'],
                'price_close': ['mae', 'mape'],
                'direction': ['accuracy', 'AUC']
            }
        )
        
        return model

    def _create_callbacks(self) -> List[callbacks.Callback]:
        """Create training callbacks"""
        return [
            callbacks.EarlyStopping(
                monitor='val_loss',
                patience=10,
                restore_best_weights=True
            ),
            callbacks.ReduceLROnPlateau(
                monitor='val_loss',
                factor=0.5,
                patience=5,
                min_lr=1e-6
            ),
            callbacks.ModelCheckpoint(
                filepath=os.path.join(self.config.get('model_dir', 'models'), 
                                    'best_model.h5'),
                monitor='val_loss',
                save_best_only=True
            )
        ]

    def preprocess_data(self, data: np.ndarray) -> Tuple[np.ndarray, MinMaxScaler]:
        """Normalize the input data"""
        self.scaler = MinMaxScaler()
        return self.scaler.fit_transform(data), self.scaler

    def create_sequences(self, data: np.ndarray) -> Tuple[np.ndarray, np.ndarray]:
        """Create sequences for training"""
        sequences = []
        targets = []
        
        for i in range(len(data) - self.sequence_length):
            seq = data[i:(i + self.sequence_length)]
            target = data[i + self.sequence_length]
            sequences.append(seq)
            targets.append(target)
            
        return np.array(sequences), np.array(targets)

    def train(self, X_train: np.ndarray, y_train: np.ndarray, 
             X_val: np.ndarray, y_val: np.ndarray, 
             batch_size: int = 32, epochs: int = 100) -> None:
        """Train the model"""
        self.model = self._create_model()
        
        self.history = self.model.fit(
            X_train,
            {
                'price_high': y_train[:, 0],
                'price_low': y_train[:, 1],
                'price_close': y_train[:, 2],
                'direction': y_train[:, 3]
            },
            validation_data=(
                X_val,
                {
                    'price_high': y_val[:, 0],
                    'price_low': y_val[:, 1],
                    'price_close': y_val[:, 2],
                    'direction': y_val[:, 3]
                }
            ),
            batch_size=batch_size,
            epochs=epochs,
            callbacks=self.callbacks,
            verbose=1
        )

    def predict(self, X: np.ndarray) -> Dict[str, np.ndarray]:
        """Make predictions"""
        if self.model is None:
            raise ValueError("Model not trained yet")
            
        predictions = self.model.predict(X)
        return {
            'price_high': self.scaler.inverse_transform(predictions[0]),
            'price_low': self.scaler.inverse_transform(predictions[1]),
            'price_close': self.scaler.inverse_transform(predictions[2]),
            'direction': predictions[3]
        }

    def save(self, path: str) -> None:
        """Save the model and scaler"""
        if self.model is None:
            raise ValueError("No model to save")
            
        model_path = os.path.join(path, 'model.h5')
        scaler_path = os.path.join(path, 'scaler.pkl')
        
        self.model.save(model_path)
        with open(scaler_path, 'wb') as f:
            pickle.dump(self.scaler, f)

    def load(self, path: str) -> None:
        """Load the model and scaler"""
        model_path = os.path.join(path, 'model.h5')
        scaler_path = os.path.join(path, 'scaler.pkl')
        
        self.model = models.load_model(model_path)
        with open(scaler_path, 'rb') as f:
            self.scaler = pickle.load(f)
