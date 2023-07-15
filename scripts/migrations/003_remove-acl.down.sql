START TRANSACTION ;

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

INSERT INTO `z42`.`ACL`
WITH res AS (
    WITH z AS (SELECT `z42`.`User`.`Id` AS User_Id, `z42`.`Zone`.`Resource_Id` FROM `z42`.`User` INNER JOIN `z42`.`Zone` ON `z42`.`User`.`Id` = `z42`.`Zone`.`User_Id`),
         l AS (SELECT z.User_Id, `z42`.`Location`.`Resource_Id` FROM z INNER JOIN `z42`.`Location` ON z.Resource_Id = `z42`.`Location`.`Zone_Id`),
         r AS (SELECT l.User_Id, `z42`.`RecordSet`.`Resource_Id` FROM l INNER JOIN `z42`.`RecordSet` ON l.Resource_Id = `z42`.`RecordSet`.`Location_Id`)
    SELECT * FROM z UNION SELECT * FROM l UNION SELECT * FROM r)
SELECT 1, 1, 1, 1, 1, res.Resource_Id, res.User_Id FROM res;

COMMIT ;
