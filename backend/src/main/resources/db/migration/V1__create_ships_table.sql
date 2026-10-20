CREATE TABLE IF NOT EXISTS ships (
    ship_id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    tier INT NOT NULL,
    roman_tier VARCHAR(10) NOT NULL,
    ship_class VARCHAR(50) NOT NULL,
    nation VARCHAR(50) NOT NULL,
    icon_url VARCHAR(1024),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ships_name ON ships(name);
CREATE INDEX IF NOT EXISTS idx_ships_tier ON ships(tier);
