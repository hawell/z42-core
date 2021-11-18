-- MySQL Workbench Forward Engineering

SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

START TRANSACTION ;

-- -----------------------------------------------------
-- Schema z42
-- -----------------------------------------------------
CREATE SCHEMA IF NOT EXISTS `z42` ;
USE `z42` ;

ALTER DATABASE `z42` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

-- -----------------------------------------------------
-- Table `z42`.`User`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`User` (
                                            `Id` CHAR(36) NOT NULL,
                                            `Email` VARCHAR(100) NOT NULL,
                                            `Password` VARCHAR(600) NOT NULL,
                                            `Status` ENUM('active', 'disabled', 'pending') NOT NULL,
                                            PRIMARY KEY (`Id`),
                                            UNIQUE INDEX `Email_UNIQUE` (`Email` ASC) VISIBLE)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Resource`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`Resource` (
                                                `Id` CHAR(36) NOT NULL,
                                                PRIMARY KEY (`Id`))
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Zone`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`Zone` (
                                            `Name` VARCHAR(256) NOT NULL,
                                            `CNameFlattening` TINYINT NOT NULL,
                                            `Dnssec` TINYINT NOT NULL,
                                            `Enabled` TINYINT NOT NULL,
                                            `Resource_Id` CHAR(36) NOT NULL,
                                            `User_Id` CHAR(36) NOT NULL,
                                            UNIQUE INDEX `Name_UNIQUE` (`Name` ASC) VISIBLE,
                                            PRIMARY KEY (`Resource_Id`),
                                            INDEX `fk_Zone_User_idx` (`User_Id` ASC) VISIBLE,
                                            CONSTRAINT `fk_Zone_Resource`
                                                FOREIGN KEY (`Resource_Id`)
                                                    REFERENCES `z42`.`Resource` (`Id`)
                                                    ON DELETE CASCADE
                                                    ON UPDATE NO ACTION,
                                            CONSTRAINT `fk_Zone_User`
                                                FOREIGN KEY (`User_Id`)
                                                    REFERENCES `z42`.`User` (`Id`)
                                                    ON DELETE CASCADE
                                                    ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Location`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`Location` (
                                                `Name` VARCHAR(256) NOT NULL,
                                                `Enabled` TINYINT NOT NULL,
                                                `Resource_Id` CHAR(36) NOT NULL,
                                                `Zone_Id` CHAR(36) NOT NULL,
                                                PRIMARY KEY (`Resource_Id`),
                                                INDEX `fk_Location_Zone_idx` (`Zone_Id` ASC) VISIBLE,
                                                UNIQUE INDEX `Zone_Name_UNIQUE` (`Zone_Id` ASC, `Name` ASC) VISIBLE,
                                                CONSTRAINT `fk_Location_Resource`
                                                    FOREIGN KEY (`Resource_Id`)
                                                        REFERENCES `z42`.`Resource` (`Id`)
                                                        ON DELETE CASCADE
                                                        ON UPDATE NO ACTION,
                                                CONSTRAINT `fk_Location_Zone`
                                                    FOREIGN KEY (`Zone_Id`)
                                                        REFERENCES `z42`.`Zone` (`Resource_Id`)
                                                        ON DELETE CASCADE
                                                        ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`RecordSet`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`RecordSet` (
                                                 `Type` ENUM('a', 'aaaa', 'aname', 'caa', 'cname', 'ds', 'mx', 'ns', 'ptr', 'srv', 'tlsa', 'txt') NOT NULL,
                                                 `Value` JSON NULL,
                                                 `Enabled` TINYINT NOT NULL,
                                                 `Resource_Id` CHAR(36) NOT NULL,
                                                 `Location_Id` CHAR(36) NOT NULL,
                                                 PRIMARY KEY (`Resource_Id`),
                                                 INDEX `fk_RecordSet_Location_idx` (`Location_Id` ASC) VISIBLE,
                                                 UNIQUE INDEX `Location_Type_UNIQUE` (`Type` ASC, `Location_Id` ASC) VISIBLE,
                                                 CONSTRAINT `fk_RecordSet_Resource`
                                                     FOREIGN KEY (`Resource_Id`)
                                                         REFERENCES `z42`.`Resource` (`Id`)
                                                         ON DELETE CASCADE
                                                         ON UPDATE NO ACTION,
                                                 CONSTRAINT `fk_RecordSet_Location`
                                                     FOREIGN KEY (`Location_Id`)
                                                         REFERENCES `z42`.`Location` (`Resource_Id`)
                                                         ON DELETE CASCADE
                                                         ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Verification`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`Verification` (
                                                    `Code` VARCHAR(100) NOT NULL,
                                                    `Type` ENUM('signup', 'recover') NOT NULL,
                                                    `User_Id` CHAR(36) NOT NULL,
                                                    UNIQUE INDEX `Code_UNIQUE` (`Code` ASC) VISIBLE,
                                                    INDEX `fk_Verification_User_idx` (`User_Id` ASC) VISIBLE,
                                                    PRIMARY KEY (`User_Id`, `Type`),
                                                    CONSTRAINT `fk_Verification_User`
                                                        FOREIGN KEY (`User_Id`)
                                                            REFERENCES `z42`.`User` (`Id`)
                                                            ON DELETE CASCADE
                                                            ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`ACL`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`ACL` (
                                           `CanRead` TINYINT NOT NULL,
                                           `CanList` TINYINT NOT NULL,
                                           `CanEdit` TINYINT NOT NULL,
                                           `CanInsert` TINYINT NOT NULL,
                                           `CanDelete` TINYINT NOT NULL,
                                           `Resource_Id` CHAR(36) NOT NULL,
                                           `User_Id` CHAR(36) NOT NULL,
                                           PRIMARY KEY (`Resource_Id`, `User_Id`),
                                           INDEX `fk_ACL_User_idx` (`User_Id` ASC) VISIBLE,
                                           INDEX `fk_ACL_Resource_idx` (`Resource_Id` ASC) VISIBLE,
                                           CONSTRAINT `fk_ACL_Resource`
                                               FOREIGN KEY (`Resource_Id`)
                                                   REFERENCES `z42`.`Resource` (`Id`)
                                                   ON DELETE CASCADE
                                                   ON UPDATE NO ACTION,
                                           CONSTRAINT `fk_ACL_User`
                                               FOREIGN KEY (`User_Id`)
                                                   REFERENCES `z42`.`User` (`Id`)
                                                   ON DELETE CASCADE
                                                   ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Events`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`Events` (
                                              `Revision` INT NOT NULL AUTO_INCREMENT,
                                              `ZoneId` CHAR(36) NOT NULL,
                                              `Type` ENUM('add_zone', 'update_zone', 'delete_zone', 'import_zone', 'add_location', 'update_location', 'delete_location', 'add_record', 'update_record', 'delete_record') NOT NULL,
                                              `Value` JSON NULL,
                                              PRIMARY KEY (`Revision`),
                                              INDEX `zone_id` (`ZoneId` ASC) VISIBLE)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`SOA`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`SOA` (
                                           `TTL` INT NOT NULL,
                                           `NS` VARCHAR(256) NOT NULL,
                                           `MBox` VARCHAR(256) NOT NULL,
                                           `Refresh` INT NOT NULL,
                                           `Retry` INT NOT NULL,
                                           `Expire` INT NOT NULL,
                                           `MinTTL` INT NOT NULL,
                                           `Serial` INT NOT NULL,
                                           `Zone_Id` CHAR(36) NOT NULL,
                                           PRIMARY KEY (`Zone_Id`),
                                           CONSTRAINT `fk_SOA_Zone`
                                               FOREIGN KEY (`Zone_Id`)
                                                   REFERENCES `z42`.`Zone` (`Resource_Id`)
                                                   ON DELETE CASCADE
                                                   ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Keys`
-- -----------------------------------------------------
CREATE TABLE IF NOT EXISTS `z42`.`Keys` (
                                            `KSK_Private` VARCHAR(2048) NOT NULL,
                                            `KSK_Public` VARCHAR(512) NOT NULL,
                                            `ZSK_Private` VARCHAR(2048) NOT NULL,
                                            `ZSK_Public` VARCHAR(512) NOT NULL,
                                            `DS` VARCHAR(256) NOT NULL,
                                            `Zone_Id` CHAR(36) NOT NULL,
                                            PRIMARY KEY (`Zone_Id`),
                                            CONSTRAINT `fk_dnssec_Zone`
                                                FOREIGN KEY (`Zone_Id`)
                                                    REFERENCES `z42`.`Zone` (`Resource_Id`)
                                                    ON DELETE CASCADE
                                                    ON UPDATE NO ACTION)
    ENGINE = InnoDB;


SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;

COMMIT ;
