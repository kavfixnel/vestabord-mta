# Vestabord MTA

A small script that reads the MTA API for upcoming subway trains
and displays when the next 1 or 2 trains will arrive

## mta/

The mta/ folder holds a small shim that is able to read the gtfs
feed for the MTA Subways. Docs here https://api.mta.info/#/subwayRealTimeFeeds

## Requirements

In order to talk to the Vestabord, a Vestaboard API token needs to be created
and saved to .env under the key VESTABORD_TOKEN. This will allow the script
to talk to https://cloud.vestaboard.com/

## help

The script has some basic settings that can change the behavior

```
$ go run . --help
vestaboard - write to a Vestaboard Note

usage:
  vestaboard                         refresh L train arrivals every 30s
  vestaboard l [-interval 1m]        same as above
  vestaboard l -once                 update once and exit
  vestaboard l -print                preview without sending to board
  vestaboard send <msg>              send a custom message

flags:
  -env .env       path to .env file (default: .env)
  -forced         send during quiet hours
  -interval 30s   refresh frequency (minimum 15s)
  -once           update once and exit
  -print          print times without sending
```
