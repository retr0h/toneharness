# Fetch a record, and prove it is the one

## Name it in the manifest first

The entry goes into `resources/music/<instrument>/<player>/corpus.yaml` before
anything is downloaded, because the link is both the evidence and what the
download reads. Written first, the record measured is the one the evidence names.

```yaml
  - track: silly-love-songs
    url: https://open.spotify.com/track/…
    year: 1976
    band: Wings
    genres: [rock, pop]
    genres_by: llm
```

`track` is the source file's own name without its extension, and the stem directory
takes the same name. A name matching nothing is reported, and so is a stray stem.

**Every entry carries a `url` and a `year`**, and the manifest is refused without a
link where it is read rather than years later. Three players once got three tracks
each and no links, and once the audio was gone nobody could refetch them or check
which takes had been measured. An album name does not stand in: it does not say
which of several tracks with that title was meant.

`band` and `genres` sit on the record rather than on the player, because a career
spans both, and a track takes as many genres as are true of it. `genres_by` says
who decided them: `llm` where a model labelled the track, which is the fast path,
and `person` where somebody said so or overruled a model. A record naming genres
without it does not read, because a guess and a checked answer look identical once
both are a word in a list, and a genre can clear its threshold entirely on
guesses.

## Use the link, never a search

Asked for a song that does not exist, spotdl downloaded an unrelated track rather
than failing. A real title gets the same treatment and a wrong match is harder to
spot: one search here returned two links that were both a different song, and one
"Welcome to Paradise" was a live broadcast. So check the link first.
`uvx spotdl save "<url>" --save-file song.spotdl` prints the artist, album, year
and duration Spotify holds, which is what tells a search result from the record you
wanted. Then download under the manifest's name, copying the url out of the entry
rather than finding it a second time:

```bash
mise exec -- just record resources/music/bass/paul-mccartney silly-love-songs \
  https://open.spotify.com/track/…
```

The recipe fails if no file appears; do not read success from spotdl's output. Where
the audio cannot be found, a YouTube link may be passed ahead of the Spotify one and
written into the entry's `source`: `url` stays the evidence a rig quotes, `source` is
only the downloader's way back. Never pass a YouTube link alone, or spotdl searches
Spotify for the video's title and tags the file as another artist's song.

## Check it is the recording

A wrong file measures without complaint, so compare lengths:

```bash
ffprobe -v error -show_entries format=duration <file>
ffmpeg -hide_banner -i <file> -af silencedetect=n=-45dB:d=1 -f null - 2>&1 \
  | grep silence_
```

Subtract the trailing silence from the file's length and it should land within a few
seconds of what Spotify holds. Lyric videos pad the end. A difference the silence
does not explain is a different recording: delete it and fetch again. A compilation
tag is fine where the compilation reuses the take.
