CREATE TABLE scans (
    ip INET NOT NULL,
    port INT NOT NULL,
    service TEXT NOT NULL,
    timestamp BIGINT NOT NULL,
    data TEXT NOT NULL,
    PRIMARY KEY (ip, port, service)
);