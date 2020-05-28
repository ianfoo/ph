# ph

Show the current song playing at [JEMP Radio](https://jempradio.com), or the
last N songs, up to the maximum number stored in the station's status' history.

When showing the current song, the elapsed time since the song has started will
be shown, and if the song is a Phish song and the title contains a date, a link
to the show on [Phish.in](https://phish.in) will be shown. Note that the
rendering of a Phish.in link does not guarantee that the show is actually
available on Phish.in. Phish.in just has predictable URLs for shows, so it is
easy to create what would be the correct URL if the show is available.

Example output:
```
❯ ph
Phish - Mercury>thru>Death Don't...(7-14-19) (started 31m28s ago)
https://phish.in/2019-07-14
```

When showing the last N songs, only the titles will be shown:
```
❯ ph -last 5
Phish - Mercury>thru>Death Don't...(7-14-19)
Phish - Steam (2-20-20)
Beatles - Dear Prudence
Phish - Space Oddity>Antelope (6-24-16)
Tom Petty - You Don't Know How It Feels
```

## Notes

You will need [Go](https://golang.org) to build or run this. You can install
this as a binary with `go install .` run in this working directory.
