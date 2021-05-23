USE `z42` ;

DELETE FROM User;

ALTER TABLE User AUTO_INCREMENT = 1;
ALTER TABLE Zone AUTO_INCREMENT = 1;
ALTER TABLE Location AUTO_INCREMENT = 1;
ALTER TABLE RecordSet AUTO_INCREMENT = 1;

INSERT INTO User(Id, Email, Password, Status) VALUES (uuid(), 'user1', 'user1', 'active');
SELECT Id INTO @user1 FROM User WHERE Email = 'user1';

INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (uuid(), 'zone1.com.', false, false, true, @user1);
SELECT Id INTO @zone1 FROM Zone WHERE Name = 'zone1.com.';
INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (uuid(), 'zone2.com.', false, false, true, @user1);
SELECT Id INTO @zone2 FROM Zone WHERE Name = 'zone2.com.';
INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (uuid(), 'zone3.com.', false, false, true, @user1);
SELECT Id INTO @zone3 FROM Zone WHERE Name = 'zone3.com.';

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'www', true, @zone1);
SELECT Id INTO @zone1_www FROM Location WHERE Name = 'www' AND Zone_Id = @zone1;
INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'a', true, @zone1);
SELECT Id INTO @zone1_a FROM Location WHERE Name = 'a' AND Zone_Id = @zone1;

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'www', true, @zone2);
SELECT Id INTO @zone2_www FROM Location WHERE Name = 'www' AND Zone_Id = @zone2;
INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'a', true, @zone2);
SELECT Id INTO @zone2_a FROM Location WHERE Name = 'a' AND Zone_Id = @zone2;

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'www', true, @zone3);
SELECT Id INTO @zone3_www FROM Location WHERE Name = 'www' AND Zone_Id = @zone3;
INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'a', true, @zone3);
SELECT Id INTO @zone3_a FROM Location WHERE Name = 'a' AND Zone_Id = @zone3;

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}', true, @zone1_www);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 www txt"}]}', true, @zone1_www);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"2.3.4.5"}]}', true, @zone1_a);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 a txt"}]}', true, @zone1_a);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"3.4.5.6"}]}', true, @zone2_www);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone2 www txt"}]}', true, @zone2_www);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"4.5.6.7"}]}', true, @zone2_a);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone2 a txt"}]}', true, @zone2_a);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"5.6.7.8"}]}', true, @zone3_www);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone3 www txt"}]}', true, @zone3_www);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"6.7.8.9"}]}', true, @zone3_a);
INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone3 a txt"}]}', true, @zone3_a);
