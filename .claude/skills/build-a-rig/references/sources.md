# Where to look, and what not to accept

Search a list, not the open web. The failure this exists to prevent is a
plausible URL off a search page pasted in as though somebody had read it.

## The list, in rough order of how often each settles something

| Source                                                            | Good for                                                                                            |
| ----------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| `web.archive.org`                                                 | dead magazines. `bassplayer.com` no longer exists and its interviews are the single richest seam     |
| Forums, searched first: talkbass.com, reddit, rickresource.com    | people who were there. A Rickenbacker tech who had McCartney's bass on his bench answered what no magazine had |
| Fan transcript archives: `2112.net` for Rush, `ram.org` for Primus | full bodies of print articles that exist nowhere else online                                        |
| Estate and luthier sites: `jacopastorius.com`, `ctbasses.com`     | first-party, and often build dates that settle which instrument was on which record                 |
| Discogs and Reverb JSON APIs                                      | sleeve credits, which are the label's own statement of who played what                              |
| premierguitar.com, guitarworld.com, mixonline.com                 | rundowns and session pieces, and reprints of dead magazines                                         |
| Maker artist pages, archived                                      | endorsements, where the capture date brackets the era                                               |
| Wikipedia                                                         | **never as a source.** Only to find the print reference underneath a claim                          |

Two that are not on it:

**equipboard.com looks like a source and is not.** It scrapes Wikipedia, says so
in its own text, and tags its own entries "Unverified".

**A search engine** is how you find a page on this list, not a source in itself.

## Read the forums with the tooling

```bash
mise exec -- just forum-search "geddy lee rickenbacker tone" Bass
mise exec -- just forum "https://www.talkbass.com/threads/…"
```

`forum` reads a thread as numbered posts each carrying its author, which is what
lets you cite the person rather than the thread.

Both sites refuse an ordinary fetch and they refuse it differently, which is why
one script handles both. TalkBass checks the TLS handshake, so no user agent gets
past its challenge; Reddit blocks its JSON API to logged-out readers however you
ask, but serves RSS to a browser user agent and refuses the impersonated one
TalkBass needs. **Sending both to both fails both.**

**A 429 from Reddit means wait, not that the thread is empty.** It throttles a
logged-out reader to roughly a request a minute. Try again rather than recording
that nothing was found: an empty answer and an absent source look identical, and
telling them apart is the whole job.

For TalkBass, find threads with a search engine and the site in the query, then
**open every thread you intend to cite**.

## Why opening it is not a formality

Every forum citation here before the tooling existed was entered from a search
result and never opened. Reading them found:

- **A quote that does not appear in the thread at all.** The McCartney tone
  thread was cited for "boosts low mids and rolls off the treble". The word
  "boost" appears zero times across its 33 posts.
- **A claim changed in the copying.** A thread says "a bunch of Ampeg SVT amps";
  it was cited as an SVT-810 cabinet, which is a different object, and five
  period sources say he used neither.
- **A thread about the wrong era.** One McCartney citation came from a thread
  whose first post asks about "the 1960's", cited in a rig covering 1975 to 1979.

## Cite the person, not the thread

A thread holds the best and the worst evidence here, often on the same page. One
rickresource.com thread carries the tech who restrung the instrument:

> When I worked on his bass in the mid 1970's he did NOT have rotosound strings
> of any kind on it. They were flats but not rickenbacker or roto's.

and, in the post directly above it, somebody guessing:

> Rotosound's are a possibility but I don't know.

A post counts when a named person with first-hand access is talking about
something they did or saw. It does not count when it is opinion, however
confident. Say which one you have.

Quote the post you are relying on into `note`, and name what makes the person
worth believing: the tech above is evidence because he had the instrument on his
bench, not because he posted confidently. **A thread that turns out to be
somebody guessing is a thread with nothing in it**, and the honest thing is to
leave the claim unsourced rather than dress the guess up in a URL.

## Three ways a citation passes a careless check

**A search result is not a source.** A search summary blends several pages and
their comment sections into one answer, and it will attribute a claim to a page
that does not make it. Four of the first eight rigs here cited something that was
not on the page once somebody opened it.

**A 200 is not verification.** One rig cited a bassplayer.com article for its
amplifier. The URL still answers 200, because the whole site now redirects to a
section index on another domain, and the article is gone. Checking that a link
resolves catches a 404 and nothing else.

**Splitting a claim does not split its evidence.** Turning "thumping low end
under a hard top" into `loose-low-end` and `bright` left both citing the rundown
that produced the sentence — and that rundown describes the low end and says
nothing about the top. One half is usually unsourced, and mechanically copying
the citation onto both is how a guess acquires a URL. Sharing a page between two
terms is fine when each quotes a different sentence from it. Sharing a sentence
is not.

## Readable is not citable

The `forum` command reads both sites, so an agent citing one has no excuse for
not opening it. A `caveat` saying nobody checked is for a page that genuinely
would not load, and it names what failed.

What the tooling does not fix is that most of a thread is people talking. One
Reddit thread on a well-known bassist returns seventeen posts and **not one is
evidence about gear**: they are enthusiasts discussing his playing, which is
worth reading and cites nothing. The rule above still decides it. A named person
with first-hand access describing what they did or saw is a source; everybody
else in the thread is company.

`video` ranks low on purpose: a stage seen from forty feet says little about
which head was on it, and the description under a clip is whatever the uploader
typed. Cite it for a word or for the technique, where hearing or seeing it is the
point, and say in the note that is what it is for.
