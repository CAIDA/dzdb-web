## TODOs from KEEP

 * add contact / about page
 * Need TLD zone page to list current number of domains (and history graph)
 * reverse index on NS
 * log 404s and other failures to app log


update templates so that first/last seen are in a box or something...

use envirotment vars instead of config.ini
docker^^

## Stats

Total zones
actively tracked zones
total tracked zones

## Zones
pi chart?
domain history per zone (ggraph)

## NS
counts of domains
-- could use list returned for < 100....
top NS

## Feeds
all


## Domains
whois?


## IPs
some statS on index (total ip4/6)?

# all
 pagnation for all pages via api?

# metadata tables


## new table
vdz=# \d nameserver_metadata 
  Table "public.nameserver_metadata"
  ┌───────────────┬─────────┬───────────┐
  │    Column     │  Type   │ Modifiers │
  ├───────────────┼─────────┼───────────┤
  │ nameserver_id │ bigint  │ not null  │
  │ first_seen    │ date    │           │
  │ last_seen     │ date    │           │
  │ domains       │ integer │           │
  │ archive       │ integer │           │
  └───────────────┴─────────┴───────────┘
  Indexes:
      "nameserver_metadata_pkey" PRIMARY KEY, btree (nameserver_id)

### queries
-- add first data
insert into nameserver_metadata (nameserver_id, first_seen) select nameserver_id, min(coalesce(first_seen,'12-27-1990')) first_seen from domains_nameservers group by nameserver_id;
-- set nulls
update nameserver_metadata set first_seen = NULL where first_seen = '1990-12-27'; 
-- insert remaining IDs
insert into nameserver_metadata (nameserver_id) SELECT id from nameservers where not exists (select nameserver_id from nameserver_metadata where nameserver_id = id);
-- add last data
create TABLE last_pp as select nameserver_id, max(last_seen) from domains_nameservers where last_seen is not null group by nameserver_id;
create table last_null as select distinct nameserver_id from domains_nameservers where last_seen is null; 
delete from last_pp where nameserver_id in (select * from last_null);
drop table last_null;
update nameserver_metadata set last_seen = max from last_pp where last_pp.nameserver_id = nameserver_metadata.nameserver_id;
drop table last_pp;
## edit import to add these fields
### first_seen
update whenever inserting new NS
### last_seen


## import_delta_counts
does not need to be public, keep internal
┌───────────┬────────┬────────┬───────┬───────┬──────────┬──────────┐
│ import_id │ old_ns │ new_ns │ old_a │ new_a │ old_aaaa │ new_aaaa │
├───────────┼────────┼────────┼───────┼───────┼──────────┼──────────┤
│      9101 │      0 │      2 │     0 │     0 │        0 │        0 │
│      9103 │     42 │     47 │     1 │     1 │        0 │        0 │
│      9102 │   8432 │   9081 │   182 │   204 │        0 │        0 │
│      9099 │  65320 │  72346 │   758 │   893 │       11 │        7 │
│      9095 │ 550804 │ 616907 │  4061 │  5305 │        4 │        8 │
└───────────┴────────┴────────┴───────┴───────┴──────────┴──────────┘

## import_zone_counts
used to generate weighted_coutns, should be shown on import data page
┌────────────┬─────────┬─────────┬─────────┬───────────┬─────┬───────┬─────┐
│    date    │ zone_id │ domains │ records │ import_id │ new │ moved │ old │
├────────────┼─────────┼─────────┼─────────┼───────────┼─────┼───────┼─────┤
│ 2016-02-25 │ 2048923 │       2 │      26 │    393203 │   2 │     0 │   0 │
│ 2016-02-25 │ 2048921 │       2 │      26 │    392588 │   2 │     0 │   0 │
│ 2016-02-25 │ 2048919 │       2 │      17 │    392816 │   0 │     0 │   0 │
│ 2016-02-25 │ 2048916 │       2 │      17 │    392900 │   0 │     0 │   0 │
│ 2016-02-25 │ 2048914 │       2 │      17 │    392612 │   0 │     0 │   0 │
└────────────┴─────────┴─────────┴─────────┴───────────┴─────┴───────┴─────┘

## weighted_coutns
use for gloabal stats in zones and total stats
┌────────────┬─────────┬──────────┬──────┬───────┬──────┐
│    date    │ zone_id │ domains  │ old  │ moved │ new  │
├────────────┼─────────┼──────────┼──────┼───────┼──────┤
│ 2013-10-07 │       6 │  1804660 │ 1649 │  1999 │ 1996 │
│ 2013-10-07 │       2 │      318 │    0 │     0 │    0 │
│ 2013-10-07 │       8 │ 10359889 │ 6043 │ 11520 │ 8004 │
│ 2013-10-07 │       7 │  2591275 │ 1912 │  2679 │ 2701 │
│ 2013-10-07 │     183 │  1155958 │  711 │  1941 │ 1686 │
└────────────┴─────────┴──────────┴──────┴───────┴──────┘


