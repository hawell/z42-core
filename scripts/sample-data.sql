USE `z42` ;

DELETE FROM User;

ALTER TABLE User AUTO_INCREMENT = 1;
ALTER TABLE Zone AUTO_INCREMENT = 1;
ALTER TABLE Location AUTO_INCREMENT = 1;
ALTER TABLE RecordSet AUTO_INCREMENT = 1;

INSERT INTO User(Email, Password) VALUES ('user1', 'user1');
SELECT Id INTO @user1 FROM User WHERE Email = 'user1';

INSERT INTO Zone(Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES ('zone1.com.', false, false, true, @user1);
SELECT Id INTO @zone1 FROM Zone WHERE Name = 'zone1.com.';
INSERT INTO Zone(Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES ('zone2.com.', false, false, true, @user1);
SELECT Id INTO @zone2 FROM Zone WHERE Name = 'zone2.com.';
INSERT INTO Zone(Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES ('zone3.com.', false, false, true, @user1);
SELECT Id INTO @zone3 FROM Zone WHERE Name = 'zone3.com.';

INSERT INTO Location(Name, Enabled, Zone_Id) VALUES ('www', true, @zone1);
SELECT Id INTO @zone1_www FROM Location WHERE Name = 'www';

INSERT INTO RecordSet(Type, Value, Enabled, Location_Id) VALUES ('a', '{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}', true, @zone1_www)
