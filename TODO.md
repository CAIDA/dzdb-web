# from KEEP

 * Need TLD zone page to list current number of domains (and history graph)
 * reverse index on NS
 * log 404s and other failures to app log

update templates so that first/last seen are in a box or something...

## Stats
Total zones
actively tracked zones
total tracked zones

## Zones
pi chart?
domain history per zone (graph)

## NS
counts of domains
-- could use list returned for < 100....
top NS

## Feeds
all

## IPs
some statS on index (total ip4/6)?

# all
 pagination for all pages via api?

# metadata tables


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
used to generate weighted_counts, should be shown on import data page
┌────────────┬─────────┬─────────┬─────────┬───────────┬─────┬───────┬─────┐
│    date    │ zone_id │ domains │ records │ import_id │ new │ moved │ old │
├────────────┼─────────┼─────────┼─────────┼───────────┼─────┼───────┼─────┤
│ 2016-02-25 │ 2048923 │       2 │      26 │    393203 │   2 │     0 │   0 │
│ 2016-02-25 │ 2048921 │       2 │      26 │    392588 │   2 │     0 │   0 │
│ 2016-02-25 │ 2048919 │       2 │      17 │    392816 │   0 │     0 │   0 │
│ 2016-02-25 │ 2048916 │       2 │      17 │    392900 │   0 │     0 │   0 │
│ 2016-02-25 │ 2048914 │       2 │      17 │    392612 │   0 │     0 │   0 │
└────────────┴─────────┴─────────┴─────────┴───────────┴─────┴───────┴─────┘

## weighted_counts
use for global stats in zones and total stats
┌────────────┬─────────┬──────────┬──────┬───────┬──────┐
│    date    │ zone_id │ domains  │ old  │ moved │ new  │
├────────────┼─────────┼──────────┼──────┼───────┼──────┤
│ 2013-10-07 │       6 │  1804660 │ 1649 │  1999 │ 1996 │
│ 2013-10-07 │       2 │      318 │    0 │     0 │    0 │
│ 2013-10-07 │       8 │ 10359889 │ 6043 │ 11520 │ 8004 │
│ 2013-10-07 │       7 │  2591275 │ 1912 │  2679 │ 2701 │
│ 2013-10-07 │     183 │  1155958 │  711 │  1941 │ 1686 │
└────────────┴─────────┴──────────┴──────┴───────┴──────┘
