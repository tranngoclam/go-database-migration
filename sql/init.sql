CREATE SCHEMA IF NOT EXISTS `auth` DEFAULT CHARACTER SET utf8mb4;
CREATE TABLE IF NOT EXISTS `auth`.`users`
(
    id         BIGINT UNSIGNED AUTO_INCREMENT NOT NULL,
    full_name  VARCHAR(511),
    address    VARCHAR(511),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id)
);

INSERT INTO `auth`.`users` (full_name, address)
VALUES ('John Doe', 'Singapore');
