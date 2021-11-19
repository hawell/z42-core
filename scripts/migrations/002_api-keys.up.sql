START TRANSACTION ;

CREATE TABLE IF NOT EXISTS `z42`.`APIKeys` (
                                               `Name` VARCHAR(256) NOT NULL,
                                               `Scope` ENUM('acme') NOT NULL,
                                               `Hash` VARCHAR(600) NOT NULL,
                                               `Enabled` TINYINT NOT NULL,
                                               `User_Id` CHAR(36) NOT NULL,
                                               `Zone_Id` CHAR(36) NOT NULL,
                                               INDEX `fk_APIKeys_User_idx` (`User_Id` ASC) VISIBLE,
                                               INDEX `fk_APIKeys_Zone_idx` (`Zone_Id` ASC) VISIBLE,
                                               INDEX `Name` (`Name` ASC) VISIBLE,
                                               UNIQUE INDEX `Name_User_UNIQUE` (`Name` ASC, `User_Id` ASC) VISIBLE,
                                               CONSTRAINT `fk_APIKeys_User`
                                                   FOREIGN KEY (`User_Id`)
                                                       REFERENCES `z42`.`User` (`Id`)
                                                       ON DELETE CASCADE
                                                       ON UPDATE NO ACTION,
                                               CONSTRAINT `fk_APIKeys_Zone`
                                                   FOREIGN KEY (`Zone_Id`)
                                                       REFERENCES `z42`.`Zone` (`Resource_Id`)
                                                       ON DELETE CASCADE
                                                       ON UPDATE NO ACTION)
    ENGINE = InnoDB;

COMMIT ;