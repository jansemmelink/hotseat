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
DROP TABLE IF EXISTS `person_nationalities`;
DROP TABLE IF EXISTS `person_parents`;
DROP TABLE IF EXISTS `persons`;

CREATE TABLE `persons` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  `surname` VARCHAR(100) NOT NULL,
  `dob` VARCHAR(10) DEFAULT NULL,
  `gender` VARCHAR(6) DEFAULT NULL,
  `email` VARCHAR(100) DEFAULT NULL,
  `phone` VARCHAR(20) DEFAULT NULL,
  UNIQUE KEY `person_id` (`id`),
  UNIQUE KEY `person_profile` (`name`,`surname`,`gender`,`dob`),
  KEY `person_name` (`name`,`surname`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

INSERT INTO `persons` VALUES
  (uuid(), 'Jan','Semmelink', '1973-11-18', 'male', 'jan.semmelink@gmail.com', '27824526299'),
  (uuid(), 'Anne-Marie','Semmelink', '1972-04-12', 'female', 'annemarie.semmelink@gmail.com', '27820913752'),
  (uuid(), 'Riaan','Semmelink', '2004-02-10', 'male', 'riaan.semmelink@gmail.com', null),
  (uuid(), 'Stefan','Semmelink', '2006-02-27', 'male', 'stefan.semmelink@gmail.com', null),
  (uuid(), 'Anja','Semmelink', '2006-02-27', 'female', 'anja.semmelink@gmail.com', null)
  ;

CREATE TABLE `person_nationalities` (
  `person_id` VARCHAR(40) NOT NULL,
  `country_id` VARCHAR(40) NOT NULL,
  `national_id` VARCHAR(100) NOT NULL,
  FOREIGN KEY (`person_id`) REFERENCES persons(`id`),
  FOREIGN KEY (`country_id`) REFERENCES countries(`id`),
  UNIQUE KEY `person_country` (`person_id`,`country_id`),
  UNIQUE KEY `country_national_id` (`country_id`,`national_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

INSERT INTO `person_nationalities` VALUES
  ((select id from persons where name="Jan" and surname="Semmelink"),(select id from countries where name="South Africa"),"7311185229089"),
  ((select id from persons where name="Anne-Marie" and surname="Semmelink"),(select id from countries where name="South Africa"),"7204120026084"),
  ((select id from persons where name="Riaan" and surname="Semmelink"),(select id from countries where name="South Africa"),"0402105666083"),
  ((select id from persons where name="Stefan" and surname="Semmelink"),(select id from countries where name="South Africa"),"0602276334086"),
  ((select id from persons where name="Anja" and surname="Semmelink"),(select id from countries where name="South Africa"),"0602271356084");

CREATE TABLE `person_parents` (
  `person_id_of_parent` VARCHAR(40) NOT NULL,
  `person_id_of_child`  VARCHAR(40) NOT NULL,
  UNIQUE KEY `person_parent` (`person_id_of_parent`,`person_id_of_child`),
  FOREIGN KEY (`person_id_of_parent`) REFERENCES persons(`id`),
  FOREIGN KEY (`person_id_of_child`) REFERENCES persons(`id`),
  CONSTRAINT `parent_child_diff` CHECK((`person_id_of_parent` <> `person_id_of_child`))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

INSERT INTO `person_parents` VALUES
  ((select id from persons where name="Jan" and surname="Semmelink"), (select id from persons where name="Riaan" and surname="Semmelink")),
  ((select id from persons where name="Jan" and surname="Semmelink"), (select id from persons where name="Stefan" and surname="Semmelink")),
  ((select id from persons where name="Jan" and surname="Semmelink"), (select id from persons where name="Anja" and surname="Semmelink")),
  ((select id from persons where name="Anne-Marie" and surname="Semmelink"), (select id from persons where name="Riaan" and surname="Semmelink")),
  ((select id from persons where name="Anne-Marie" and surname="Semmelink"), (select id from persons where name="Stefan" and surname="Semmelink")),
  ((select id from persons where name="Anne-Marie" and surname="Semmelink"), (select id from persons where name="Anja" and surname="Semmelink"));

CREATE TABLE `accounts` (
  `id` VARCHAR(40) DEFAULT (uuid()),
  `name` VARCHAR(64) NOT NULL,
  `admin` boolean DEFAULT false,
  `active` boolean DEFAULT true,
  `expiry` DATETIME DEFAULT NULL,
  UNIQUE KEY `account_id` (`id`),
  UNIQUE KEY `account_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

INSERT INTO `accounts` values
  (uuid(), "admin", true, true, NULL),
  (uuid(), "public", false, true, NULL);

CREATE TABLE `users` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `account_id` VARCHAR(40) NOT NULL,
  `username` VARCHAR(64) NOT NULL,
  `passhash` VARCHAR(40) NOT NULL,
  `admin` boolean DEFAULT false,
  `active` boolean DEFAULT true,
  `expiry` DATETIME DEFAULT NULL,
  `person_id` VARCHAR(40) DEFAULT NULL,
  UNIQUE KEY `user_id` (`id`),
  UNIQUE KEY `user_name` (`username`),
  FOREIGN KEY (`account_id`) REFERENCES accounts(`id`),
  FOREIGN KEY (`person_id`) REFERENCES persons(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

-- admin password is 
INSERT INTO `users` VALUES
(uuid(),(select id from `accounts` where name='admin'),'admin','c2ed52119def9580a09f5edcc56a99d364d5928b',true,true,NULL,NULL);

CREATE TABLE `sessions` (
  `token` VARCHAR(40) DEFAULT(uuid()) NOT NULL,
  `account_id` VARCHAR(40) NOT NULL,
  `user_id` VARCHAR(40) NOT NULL,
  `time_created` TIMESTAMP(0) NOT NULL DEFAULT now(),
  `time_updated` TIMESTAMP(0) NOT NULL DEFAULT now(),
  FOREIGN KEY (`account_id`) REFERENCES accounts(`id`),
  FOREIGN KEY (`user_id`) REFERENCES users(`id`),
  UNIQUE KEY `session_token` (`token`),
  UNIQUE KEY `session_user` (`account_id`,`user_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

--============================================

DROP TABLE IF EXISTS `adresses`;
DROP TABLE IF EXISTS `regions`;
DROP TABLE IF EXISTS `countries`;
CREATE TABLE `countries` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  `national_id_regex_pattern` VARCHAR(200) DEFAULT NULL,
  UNIQUE KEY `country_id` (`id`),
  UNIQUE KEY `country_name` (`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

INSERT INTO `countries` VALUES (uuid(), 'South Africa','[0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9][0-9]');

CREATE TABLE `regions` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `country_id` VARCHAR(40) NOT NULL,
  `name` VARCHAR(100) NOT NULL,
  `code` VARCHAR(10) NOT NULL,
  UNIQUE KEY `region_id` (`id`),
  UNIQUE KEY `region_name` (`country_id`,`name`),
  UNIQUE KEY `region_code` (`country_id`,`code`),
  FOREIGN KEY (`country_id`) REFERENCES `countries`(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

INSERT INTO `regions` VALUES
  (uuid(), (select id from countries where name="South Africa"), "Gauteng", "GP"),
  (uuid(), (select id from countries where name="South Africa"), "Kwazulu-Natal", "KZ"),
  (uuid(), (select id from countries where name="South Africa"), "Mpumalanga", "MP"),
  (uuid(), (select id from countries where name="South Africa"), "Western Cape", "WC"),
  (uuid(), (select id from countries where name="South Africa"), "Free State", "FS"),
  (uuid(), (select id from countries where name="South Africa"), "North West", "NW"),
  (uuid(), (select id from countries where name="South Africa"), "Eastern Cape", "EC"),
  (uuid(), (select id from countries where name="South Africa"), "Northern Cape", "NC"),
  (uuid(), (select id from countries where name="South Africa"), "Limpopo", "LP");


CREATE TABLE `addresses` (
  id VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  phone VARCHAR(20) DEFAULT NULL,
  street VARCHAR(100) NOT NULL,
  info VARCHAR(100) NOT NULL,
  city VARCHAR(100) NOT NULL,
  region_id VARCHAR(40) NOT NULL,
  code VARCHAR(20) NOT NULL,
  -- lat/lon ...
  UNIQUE KEY `address_id` (`id`),
  FOREIGN KEY (`region_id`) REFERENCES regions (`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;


--==================================

DROP TABLE IF EXISTS `group_members`;
DROP TABLE IF EXISTS `groups`;

CREATE TABLE `groups` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `account_id` VARCHAR(40) NOT NULL,
  `name` VARCHAR(64) NOT NULL,
  `owner_type` VARCHAR(40) NOT NULL,
  `owner_id` VARCHAR(40) NOT NULL,
  UNIQUE KEY `group_id` (`id`),
  UNIQUE KEY `group_name` (`account_id`,`name`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

CREATE TABLE `group_members` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `group_id` VARCHAR(40) NOT NULL,
  `member_type` VARCHAR(40) NOT NULL,
  `member_id` VARCHAR(40) NOT NULL,
  UNIQUE KEY `group_member` (`group_id`,`member_id`),
  FOREIGN KEY (`group_id`) REFERENCES groups(`id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;

--========================================

CREATE TABLE `events` (
  `id` VARCHAR(40) DEFAULT (uuid()) NOT NULL,
  `account_id` VARCHAR(40) NOT NULL,
  `name` VARCHAR(200) NOT NULL,
  


) ENGINE=InnoDB DEFAULT CHARSET=utf8mb3;