START TRANSACTION ;

DROP TABLE IF EXISTS `z42`.`UserZone` ;

CREATE TABLE IF NOT EXISTS `z42`.`UserZone` (
    `User_Id` CHAR(36) NOT NULL,
    `Zone_Id` CHAR(36) NOT NULL,
    `Role` ENUM('owner', 'read', 'write', 'disabled') NOT NULL,
    PRIMARY KEY (`User_Id`, `Zone_Id`),
    INDEX `fk_UserZone_Zone_idx` (`Zone_Id` ASC) VISIBLE,
    INDEX `fk_UserZone_User_idx` (`User_Id` ASC) VISIBLE,
    CONSTRAINT `fk_UserZone_User`
        FOREIGN KEY (`User_Id`)
            REFERENCES `z42`.`User` (`Id`)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION,
    CONSTRAINT `fk_UserZone_Zone`
        FOREIGN KEY (`Zone_Id`)
            REFERENCES `z42`.`Zone` (`Resource_Id`)
            ON DELETE NO ACTION
            ON UPDATE NO ACTION)
    ENGINE = InnoDB;

INSERT INTO `z42`.`UserZone` SELECT z.`User_Id`, z.`Resource_Id`, 'owner' FROM `z42`.`Zone` z;

ALTER TABLE `z42`.`Zone` DROP FOREIGN KEY `fk_Zone_User`;
ALTER TABLE `z42`.`Zone` DROP COLUMN 'User_Id';

COMMIT ;
