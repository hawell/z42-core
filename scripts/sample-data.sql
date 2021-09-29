USE `z42` ;

DELETE FROM User;

INSERT INTO User(Id, Email, Password, Status) VALUES (uuid(), 'user1@domain.com', '$2a$14$1zD30XaixpZOVb7KfFPu3O2NiU8y.T.QDFb0ztYaL7ekmLnZTG8te', 'active');
SELECT Id INTO @user1 FROM User WHERE Email = 'user1@domain.com';

SET @zone1  = uuid();
INSERT INTO Resource(Id) VALUES (@zone1);
INSERT INTO Zone(Resource_Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (@zone1, 'zone1.com.', false, false, true, @user1);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1, true, true, true, true, true, @user1);

SET @zone1_root = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_root);
INSERT INTO Location(Resource_Id, Name, Enabled, Zone_Id) VALUES (@zone1_root, '@', true, @zone1);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_root, true, true, true, true, false, @user1);

SET @zone1_www = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_www);
INSERT INTO Location(Resource_Id, Name, Enabled, Zone_Id) VALUES (@zone1_www, 'www', true, @zone1);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_www, true, true, true, true, true, @user1);

SET @zone1_a = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_a);
INSERT INTO Location(Resource_Id, Name, Enabled, Zone_Id) VALUES (@zone1_a, 'a', true, @zone1);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_a, true, true, true, true, true, @user1);

INSERT INTO SOA(TTL, NS, MBox, Refresh, Retry, Expire, MinTTL, Serial, Zone_Id) VALUES (300, 'ns1.zone1.com.', 'hostmaster.zone1.com.', 44, 55, 66, 100, 12345, @zone1);

SET @zone1_root_ns = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_root_ns);
INSERT INTO RecordSet(Resource_Id, Type, Value, Enabled, Location_Id) VALUES (@zone1_root_ns, 'ns', '{"ttl":300, "records":[{"host":"ns1.zone1.com."},{"host":"ns2.zone1.com."}]}', true, @zone1_root);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_root_ns, true, true, true, true, true, @user1);

SET @zone1_www_a = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_www_a);
INSERT INTO RecordSet(Resource_Id, Type, Value, Enabled, Location_Id) VALUES (@zone1_www_a, 'a', '{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}', true, @zone1_www);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_www_a, true, true, true, true, true, @user1);

SET @zone1_www_txt = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_www_txt);
INSERT INTO RecordSet(Resource_Id, Type, Value, Enabled, Location_Id) VALUES (@zone1_www_txt, 'txt', '{"ttl":300, "records":[{"text":"zone1 www txt"}]}', true, @zone1_www);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_www_txt, true, true, true, true, true, @user1);

SET @zone1_a_a = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_a_a);
INSERT INTO RecordSet(Resource_Id, Type, Value, Enabled, Location_Id) VALUES (@zone1_a_a, 'a', '{"ttl": 300, "records":[{"ip":"2.3.4.5"}]}', true, @zone1_a);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_a_a, true, true, true, true, true, @user1);

SET @zone1_a_txt = uuid();
INSERT INTO Resource(Id) VALUES (@zone1_a_txt);
INSERT INTO RecordSet(Resource_Id, Type, Value, Enabled, Location_Id) VALUES (@zone1_a_txt, 'txt', '{"ttl":300, "records":[{"text":"zone1 a txt"}]}', true, @zone1_a);
INSERT INTO ACL(Resource_Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_a_txt, true, true, true, true, true, @user1);
