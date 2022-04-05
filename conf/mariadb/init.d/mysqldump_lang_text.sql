GRANT ALL PRIVILEGES ON `hotseat`.* to 'hotseat'@'%' IDENTIFIED BY 'hotseat';

DROP TABLE IF EXISTS `log`;
CREATE TABLE `log` (
  `table` VARCHAR(40) NOT NULL,
  `id` VARCHAR(40) NOT NULL,
  `timestamp` TIMESTAMP(6) DEFAULT now(),
  `change` VARCHAR(40) NOT NULL,
  `details` VARCHAR(64) NOT NULL,
  UNIQUE KEY `log_id` (`table`,`id`,`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

DROP TABLE IF EXISTS `sessions`;
DROP TABLE IF EXISTS `users`;
DROP TABLE IF EXISTS `accounts`;

CREATE TABLE `accounts` (
  `id` VARCHAR(40) DEFAULT (uuid()),
  `name` varchar(64) NOT NULL,
  `admin` boolean DEFAULT false,
  `active` boolean DEFAULT true,
  `expiry` DATETIME DEFAULT NULL,
  UNIQUE KEY `account_id` (`id`),
  UNIQUE KEY `account_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

INSERT INTO `accounts` values (uuid(), "admin", true, true, NULL);

CREATE TABLE `users` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `account_id` varchar(40) NOT NULL,
  `username` varchar(64) NOT NULL,
  `passhash` varchar(40) NOT NULL,
  `admin` boolean DEFAULT false,
  `active` boolean DEFAULT true,
  `expiry` DATETIME DEFAULT NULL,
  UNIQUE KEY `user_id` (`id`),
  UNIQUE KEY `user_name` (`username`),
  FOREIGN KEY (`account_id`) REFERENCES accounts(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

LOCK TABLES `users` WRITE;
INSERT INTO `users` VALUES
(uuid(),(select id from `accounts` where name='admin'),'admin','c2ed52119def9580a09f5edcc56a99d364d5928b',true,true,NULL);
UNLOCK TABLES;


CREATE TABLE `sessions` (
  `token` VARCHAR(40) DEFAULT(uuid()) NOT NULL,
  `account_id` varchar(40) NOT NULL,
  `user_id` varchar(40) NOT NULL,
  `time_created` TIMESTAMP(0) NOT NULL DEFAULT now(),
  `time_updated` TIMESTAMP(0) NOT NULL DEFAULT now(),
  FOREIGN KEY (`account_id`) REFERENCES accounts(`id`),
  FOREIGN KEY (`user_id`) REFERENCES users(`id`),
  UNIQUE KEY `session_token` (`token`),
  UNIQUE KEY `session_user` (`account_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

--==================================

DROP TABLE IF EXISTS `group_members`;
DROP TABLE IF EXISTS `groups`;

CREATE TABLE `groups` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `account_id` varchar(40) NOT NULL,
  `name` varchar(64) NOT NULL,
  `owner_type` varchar(40) NOT NULL,
  `owner_id` varchar(40) NOT NULL,
  UNIQUE KEY `group_id` (`id`),
  UNIQUE KEY `group_name` (`account_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

CREATE TABLE `group_members` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `group_id` VARCHAR(40) NOT NULL,
  `member_type` varchar(40) NOT NULL,
  `member_id` varchar(40) NOT NULL,
  UNIQUE KEY `group_member` (`group_id`,`member_id`),
  FOREIGN KEY (`group_id`) REFERENCES groups(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;
