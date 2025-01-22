-- Drop tables in reverse order of creation to handle dependencies
DROP TABLE IF EXISTS trading_signals;
DROP TABLE IF EXISTS indicators;
DROP TABLE IF EXISTS predictions;
DROP TABLE IF EXISTS market_analysis;
DROP TABLE IF EXISTS asset_prices;
DROP TABLE IF EXISTS portfolio_performance;
DROP TABLE IF EXISTS assets;
DROP TABLE IF EXISTS portfolios;
DROP TABLE IF EXISTS users;

-- Drop extensions
DROP EXTENSION IF EXISTS "btree_gist";
DROP EXTENSION IF EXISTS "uuid-ossp";