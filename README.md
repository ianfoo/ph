# ph

Show the current song playing at [JEMP Radio](https://jempradio.com), or the
last N songs, up to the maximum number stored in the station's status' history.

When showing the current song, the elapsed time since the song has started will
be shown, and if the song is a Phish song or one of a set of other bands
commonly played, and the title contains a date, a link to the show on
[Relisten](https://relisten.net) will be shown. Note that the rendering of a
Relisten link does not guarantee that the show is actually available on to
stream. Relisten just has predictable URLs for shows, so it is easy to create
what would be the correct URL if the show is available.

Example output:
```
❯ ph
Phish - Mercury>thru>Death Don't... (Sun 14-Jul-2019) (started 31m28s ago)
https://relisten.net/phish/2019/07/14
```

The output will be compressed onto a single line to help the readability of the
list when showing history. Track histories do contain the time when the track
started playing.
```
❯ ph -last 5
 1. Phish - The Moma Dance (Mon 20-Jul-1998) - https://relisten.net/phish/1998/07/20
 2. Cream - Crossroads
 3. Phish - McGrupp And The Watchful (Thu 29-Oct-1998) - https://relisten.net/phish/1998/10/29
 4. Grateful Dead - Hell In A Bucket - Keep Your Day Job (Fri 14-Oct-1983) - https://relisten.net/grateful-dead/1983/10/14
 5. Phish - Punch You In The Eye>Reba (Thu 14-Sep-2000) - https://relisten.net/phish/2000/09/14
```

## Notes

You will need [Go](https://golang.org) to build or run this. You can install
this as a binary with `go install .` run in this working directory.

## TODO
* Scrub "www.jempradio.com - JEMP Radio" from track history
* Poll occasionally and record data to a database for analysis
