USE `z42` ;

DELETE FROM User;

INSERT INTO User(Id, Email, Password, Status) VALUES (uuid(), 'user1@domain.com', '$2a$14$1zD30XaixpZOVb7KfFPu3O2NiU8y.T.QDFb0ztYaL7ekmLnZTG8te', 'active');
SELECT Id INTO @user1 FROM User WHERE Email = 'user1@domain.com';

INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (uuid(), 'zone1.com.', false, false, true, @user1);
SELECT Id INTO @zone1 FROM Zone WHERE Name = 'zone1.com.';
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1, true, true, true, true, true, @user1);

INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (uuid(), 'zone2.com.', false, false, true, @user1);
SELECT Id INTO @zone2 FROM Zone WHERE Name = 'zone2.com.';
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2, true, true, true, true, true, @user1);

INSERT INTO Zone(Id, Name, CNameFlattening, Dnssec, Enabled, User_Id) VALUES (uuid(), 'zone3.com.', false, false, true, @user1);
SELECT Id INTO @zone3 FROM Zone WHERE Name = 'zone3.com.';
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3, true, true, true, true, true, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), '@', true, @zone1);
SELECT Id INTO @zone1_root FROM Location WHERE Name = '@' AND Zone_Id = @zone1;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_root, true, true, true, true, false, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'www', true, @zone1);
SELECT Id INTO @zone1_www FROM Location WHERE Name = 'www' AND Zone_Id = @zone1;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_www, true, true, true, true, true, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'a', true, @zone1);
SELECT Id INTO @zone1_a FROM Location WHERE Name = 'a' AND Zone_Id = @zone1;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_a, true, true, true, true, true, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), '@', true, @zone2);
SELECT Id INTO @zone2_root FROM Location WHERE Name = '@' AND Zone_Id = @zone2;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_root, true, true, true, true, false, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'www', true, @zone2);
SELECT Id INTO @zone2_www FROM Location WHERE Name = 'www' AND Zone_Id = @zone2;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_www, true, true, true, true, true, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'a', true, @zone2);
SELECT Id INTO @zone2_a FROM Location WHERE Name = 'a' AND Zone_Id = @zone2;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_a, true, true, true, true, true, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), '@', true, @zone3);
SELECT Id INTO @zone3_root FROM Location WHERE Name = '@' AND Zone_Id = @zone3;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_root, true, true, true, true, false, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'www', true, @zone3);
SELECT Id INTO @zone3_www FROM Location WHERE Name = 'www' AND Zone_Id = @zone3;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_www, true, true, true, true, true, @user1);

INSERT INTO Location(Id, Name, Enabled, Zone_Id) VALUES (uuid(), 'a', true, @zone3);
SELECT Id INTO @zone3_a FROM Location WHERE Name = 'a' AND Zone_Id = @zone3;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'soa', '{"ttl":300, "minttl":100, "mbox":"hostmaster.zone1.com.","ns":"ns1.zone1.com.","refresh":44,"retry":55,"expire":66}', true, @zone1_root);
SELECT Id INTO @zone1_root_soa FROM RecordSet WHERE Type = 'soa' AND Location_Id = @zone1_root;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_root_soa, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'ns', '{"ttl":300, "records":[{"host":"ns1.zone1.com."},{"host":"ns2.zone1.com."}]}', true, @zone1_root);
SELECT Id INTO @zone1_root_ns FROM RecordSet WHERE Type = 'ns' AND Location_Id = @zone1_root;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_root_ns, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}', true, @zone1_www);
SELECT Id INTO @zone1_www_a FROM RecordSet WHERE Type = 'a' AND Location_Id = @zone1_www;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_www_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 www txt"}]}', true, @zone1_www);
SELECT Id INTO @zone1_www_txt FROM RecordSet WHERE Type = 'txt' AND Location_Id = @zone1_www;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_www_txt, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"2.3.4.5"}]}', true, @zone1_a);
SELECT Id INTO @zone1_a_a FROM RecordSet WHERE Type = 'a' AND Location_Id = @zone1_a;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_a_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 a txt"}]}', true, @zone1_a);
SELECT Id INTO @zone1_a_a FROM RecordSet WHERE Type = 'txt' AND Location_Id = @zone1_a;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone1_a_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'soa', '{"ttl":300, "minttl":100, "mbox":"hostmaster.zone2.com.","ns":"ns1.zone2.com.","refresh":44,"retry":55,"expire":66}', true, @zone2_root);
SELECT Id INTO @zone2_root_soa FROM RecordSet WHERE Type = 'soa' AND Location_Id = @zone2_root;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_root_soa, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'ns', '{"ttl":300, "records":[{"host":"ns1.zone2.com."},{"host":"ns2.zone2.com."}]}', true, @zone2_root);
SELECT Id INTO @zone2_root_ns FROM RecordSet WHERE Type = 'ns' AND Location_Id = @zone2_root;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_root_ns, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}', true, @zone2_www);
SELECT Id INTO @zone2_www_a FROM RecordSet WHERE Type = 'a' AND Location_Id = @zone2_www;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_www_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 www txt"}]}', true, @zone2_www);
SELECT Id INTO @zone2_www_txt FROM RecordSet WHERE Type = 'txt' AND Location_Id = @zone2_www;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_www_txt, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"2.3.4.5"}]}', true, @zone2_a);
SELECT Id INTO @zone2_a_a FROM RecordSet WHERE Type = 'a' AND Location_Id = @zone2_a;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_a_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 a txt"}]}', true, @zone2_a);
SELECT Id INTO @zone2_a_a FROM RecordSet WHERE Type = 'txt' AND Location_Id = @zone2_a;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone2_a_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'soa', '{"ttl":300, "minttl":100, "mbox":"hostmaster.zone3.com.","ns":"ns1.zone3.com.","refresh":44,"retry":55,"expire":66}', true, @zone3_root);
SELECT Id INTO @zone3_root_soa FROM RecordSet WHERE Type = 'soa' AND Location_Id = @zone3_root;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_root_soa, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'ns', '{"ttl":300, "records":[{"host":"ns1.zone3.com."},{"host":"ns2.zone3.com."}]}', true, @zone3_root);
SELECT Id INTO @zone3_root_ns FROM RecordSet WHERE Type = 'ns' AND Location_Id = @zone3_root;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_root_ns, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"1.2.3.4"}]}', true, @zone3_www);
SELECT Id INTO @zone3_www_a FROM RecordSet WHERE Type = 'a' AND Location_Id = @zone3_www;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_www_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 www txt"}]}', true, @zone3_www);
SELECT Id INTO @zone3_www_txt FROM RecordSet WHERE Type = 'txt' AND Location_Id = @zone3_www;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_www_txt, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'a', '{"ttl": 300, "records":[{"ip":"2.3.4.5"}]}', true, @zone3_a);
SELECT Id INTO @zone3_a_a FROM RecordSet WHERE Type = 'a' AND Location_Id = @zone3_a;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_a_a, true, true, true, true, true, @user1);

INSERT INTO RecordSet(Id, Type, Value, Enabled, Location_Id) VALUES (uuid(), 'txt', '{"ttl":300, "records":[{"text":"zone1 a txt"}]}', true, @zone3_a);
SELECT Id INTO @zone3_a_a FROM RecordSet WHERE Type = 'txt' AND Location_Id = @zone3_a;
INSERT INTO ACL(Id, CanRead, CanList, CanEdit, CanInsert, CanDelete, User_Id) VALUES (@zone3_a_a, true, true, true, true, true, @user1);

