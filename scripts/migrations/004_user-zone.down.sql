START TRANSACTION ;

ALTER TABLE `z42`.`Zone` ADD COLUMN `User_Id` char(36);
ALTER TABLE `z42`.`Zone` ADD CONSTRAINT `fk_Zone_User` FOREIGN KEY (`User_Id`) REFERENCES `User` (`Id`) ON DELETE CASCADE;
UPDATE `z42`.`Zone` z JOIN `z42`.`UserZone` uz ON z.Resource_Id = uz.Zone_Id SET
    z.User_Id = (SELECT User_Id FROM `z42`.`UserZone` uz2 WHERE uz2.User_Id = uz.User_Id AND uz2.Zone_Id = z.Resource_Id AND uz2.Role = 'owner');
DROP TABLE IF EXISTS `z42`.`UserZone` ;

COMMIT ;
