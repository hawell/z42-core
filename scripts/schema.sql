-- MySQL Workbench Forward Engineering

SET @OLD_UNIQUE_CHECKS=@@UNIQUE_CHECKS, UNIQUE_CHECKS=0;
SET @OLD_FOREIGN_KEY_CHECKS=@@FOREIGN_KEY_CHECKS, FOREIGN_KEY_CHECKS=0;
SET @OLD_SQL_MODE=@@SQL_MODE, SQL_MODE='ONLY_FULL_GROUP_BY,STRICT_TRANS_TABLES,NO_ZERO_IN_DATE,NO_ZERO_DATE,ERROR_FOR_DIVISION_BY_ZERO,NO_ENGINE_SUBSTITUTION';

-- -----------------------------------------------------
-- Schema z42
-- -----------------------------------------------------

-- -----------------------------------------------------
-- Schema z42
-- -----------------------------------------------------
CREATE SCHEMA IF NOT EXISTS `z42` ;
USE `z42` ;

-- -----------------------------------------------------
-- Table `z42`.`User`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `z42`.`User` ;

CREATE TABLE IF NOT EXISTS `z42`.`User` (
                                            `Id` INT NOT NULL AUTO_INCREMENT,
                                            `Email` VARCHAR(100) NOT NULL,
                                            `Password` VARCHAR(600) NOT NULL,
                                            `Status` ENUM('active', 'disabled', 'pending') NOT NULL,
                                            PRIMARY KEY (`Id`),
                                            UNIQUE INDEX `Email_UNIQUE` (`Email` ASC) VISIBLE)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Zone`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `z42`.`Zone` ;

CREATE TABLE IF NOT EXISTS `z42`.`Zone` (
                                            `Id` INT NOT NULL AUTO_INCREMENT,
                                            `Name` VARCHAR(45) NOT NULL,
                                            `CNameFlattening` TINYINT NOT NULL,
                                            `Dnssec` TINYINT NOT NULL,
                                            `Enabled` TINYINT NOT NULL,
                                            `User_Id` INT NOT NULL,
                                            PRIMARY KEY (`Id`),
                                            INDEX `fk_Zone_User_idx` (`User_Id` ASC) VISIBLE,
                                            UNIQUE INDEX `Name_UNIQUE` (`Name` ASC) VISIBLE,
                                            CONSTRAINT `fk_Zone_User`
                                                FOREIGN KEY (`User_Id`)
                                                    REFERENCES `z42`.`User` (`Id`)
                                                    ON DELETE CASCADE
                                                    ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Location`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `z42`.`Location` ;

CREATE TABLE IF NOT EXISTS `z42`.`Location` (
                                                `Id` INT NOT NULL AUTO_INCREMENT,
                                                `Name` VARCHAR(45) NOT NULL,
                                                `Enabled` TINYINT NOT NULL,
                                                `Zone_Id` INT NOT NULL,
                                                PRIMARY KEY (`Id`),
                                                INDEX `fk_Location_Zone_idx` (`Zone_Id` ASC) VISIBLE,
                                                UNIQUE INDEX `Location_Zone_Unique` (`Zone_Id` ASC, `Name` ASC) VISIBLE,
                                                CONSTRAINT `fk_Location_Zone`
                                                    FOREIGN KEY (`Zone_Id`)
                                                        REFERENCES `z42`.`Zone` (`Id`)
                                                        ON DELETE CASCADE
                                                        ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`RecordSet`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `z42`.`RecordSet` ;

CREATE TABLE IF NOT EXISTS `z42`.`RecordSet` (
                                                 `Id` INT NOT NULL AUTO_INCREMENT,
                                                 `Type` ENUM('a', 'aaaa', 'cname', 'txt', 'ns', 'mx', 'srv', 'caa', 'ptr', 'tlsa', 'ds', 'aname') NOT NULL,
                                                 `Value` JSON NULL,
                                                 `Enabled` TINYINT NOT NULL,
                                                 `Location_Id` INT NOT NULL,
                                                 PRIMARY KEY (`Id`),
                                                 INDEX `fk_RecordSet_Location_idx` (`Location_Id` ASC) VISIBLE,
                                                 UNIQUE INDEX `Type_Location_Unique` (`Type` ASC, `Location_Id` ASC) VISIBLE,
                                                 CONSTRAINT `fk_RecordSet_Location`
                                                     FOREIGN KEY (`Location_Id`)
                                                         REFERENCES `z42`.`Location` (`Id`)
                                                         ON DELETE CASCADE
                                                         ON UPDATE NO ACTION)
    ENGINE = InnoDB;


-- -----------------------------------------------------
-- Table `z42`.`Verification`
-- -----------------------------------------------------
DROP TABLE IF EXISTS `z42`.`Verification` ;

CREATE TABLE IF NOT EXISTS `z42`.`Verification` (
                                                    `Code` VARCHAR(100) NOT NULL,
                                                    `Type` ENUM('signup') NULL,
                                                    `User_Id` INT NOT NULL,
                                                    PRIMARY KEY (`Code`),
                                                    UNIQUE INDEX `Code_UNIQUE` (`Code` ASC) VISIBLE,
                                                    INDEX `fk_Verification_User_idx` (`User_Id` ASC) VISIBLE,
                                                    CONSTRAINT `fk_Verification_User1`
                                                        FOREIGN KEY (`User_Id`)
                                                            REFERENCES `z42`.`User` (`Id`)
                                                            ON DELETE CASCADE
                                                            ON UPDATE NO ACTION)
    ENGINE = InnoDB;


SET SQL_MODE=@OLD_SQL_MODE;
SET FOREIGN_KEY_CHECKS=@OLD_FOREIGN_KEY_CHECKS;
SET UNIQUE_CHECKS=@OLD_UNIQUE_CHECKS;
