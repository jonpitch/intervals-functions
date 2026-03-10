# intervals-functions

functions to sync data from 3rd party sources to [intervals.icu](https://intervals.icu/) that aren't natively supported

# running

add your cronometer and intervals credentials 
```
cp .env.example .env
# update .env accordingly
```

update the intervals wellness record for yesterday:
```
go run cmd/cronometer.go
```

backfill wellness records:
```
cd cmd/backfill
go run backfill.go fromDate toDate // dates are in YYYY-MM-DD format
```

# testing

`go test ./...`

# notes

- [gocronometer](https://github.com/jrmycanady/gocronometer) doesn't work with 2FA
- running `backfill` with large date ranges can lead to throttling from cronometer
- `NETLIFY` as an environment variable is not available at runtime, only [these](https://docs.netlify.com/build/functions/environment-variables/#functions) are
- netlify functions run in UTC time
- spO2 has no 4 week window in Garmin Connect

# TODO

- move backfill to cronometer_backfill
- implement garmin backfill
- build backfill binaries to /dist