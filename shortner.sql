create table ShortLink(
  Shortened varchar(8) PRIMARY KEY,
  Original varchar(255),
  Expiry int,
  Created  datetime,
  Hits int
);