// Copyright (c) 2026 John Dewey

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to
// deal in the Software without restriction, including without limitation the
// rights to use, copy, modify, merge, publish, distribute, sublicense, and/or
// sell copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING
// FROM, OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER
// DEALINGS IN THE SOFTWARE.

package sdk

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/retr0h/toneharness/pkg/sdk/audio"
	"github.com/retr0h/toneharness/pkg/sdk/catalog"
	"github.com/retr0h/toneharness/pkg/sdk/internal/asking"
	"github.com/retr0h/toneharness/pkg/sdk/internal/attached"
	"github.com/retr0h/toneharness/pkg/sdk/internal/backup"
	"github.com/retr0h/toneharness/pkg/sdk/internal/catalogview"
	"github.com/retr0h/toneharness/pkg/sdk/internal/device"
	"github.com/retr0h/toneharness/pkg/sdk/internal/deviceslots"
	"github.com/retr0h/toneharness/pkg/sdk/internal/fileslots"
	"github.com/retr0h/toneharness/pkg/sdk/internal/musicview"
	"github.com/retr0h/toneharness/pkg/sdk/internal/presets"
	"github.com/retr0h/toneharness/pkg/sdk/internal/presetsview"
	"github.com/retr0h/toneharness/pkg/sdk/internal/rigs"
	"github.com/retr0h/toneharness/pkg/sdk/measured"
)

// Client is what a wrapper holds.
//
// One type to rally around, so a terminal, a service and a TUI reach the same
// operations the same way. Every method answers with a value and decides
// nothing about how it looks: the caller draws a table, serialises JSON or
// keeps a cursor in it, and none of those three has to know about the others.
//
// Where a device is needed the Client finds one; where it is not, the same
// Client works with no hardware attached. That is what lets a service compile
// rigs on a machine that has never seen a Helix.
//
// A Client is safe for concurrent use. Build one with New. The zero Client
// reads files and the built-in catalog, but has no bus to reach a device
// through.
type Client struct {
	opts options

	// fileFlows and deviceFlows are the slot operations, configured from opts
	// once, on first use, so a Client that did not come from New still has
	// them.
	flowsOnce   sync.Once
	fileFlows   *fileslots.Flows
	deviceFlows *deviceslots.Flows

	// mu guards cat, the catalog opened on first use.
	mu  sync.Mutex
	cat *catalog.Catalog

	// claim is held by the Session this Client has open. One slot: a
	// second Open waits for it.
	claim chan struct{}
}

// options are what New was given.
//
// Unexported, with a function per setting, so adding one changes no call
// already written.
type options struct {
	// catalog is a generated catalog. Empty means the built-in one.
	catalog string
	// device names which built-in catalog to use, for somebody whose pedal is
	// not the HX Stomp everything here was written against. Empty means that
	// one. An explicit catalog wins over it.
	device string
	// stats is measured corpus statistics. Empty means the built-in ones.
	stats string
	// setup is what the person has, which is what makes an answer fit them
	// rather than the record. Empty builds for the record.
	setup string
	// rigs is a directory of rigs. Empty means the ones that ship.
	rigs string
	// userRigs is somebody's own directory of rigs, layered over rigs.
	// Empty layers nothing.
	userRigs string
	// backupDir is where a slot's old contents go. Empty means the state
	// directory.
	backupDir string
	// capture receives each device answer a read gets. Nil keeps nothing.
	capture io.Writer
	// trace receives every USB frame in and out. Nil traces nothing.
	trace io.Writer
	// devices is the bus. Nil until New fills in USB.
	devices device.Opener
}

// Option changes how a Client works.
type Option func(*options)

// WithCatalog reads a generated catalog instead of the built-in one.
func WithCatalog(
	path string,
) Option {
	return func(o *options) { o.catalog = path }
}

// WithDevice uses the built-in catalog for a named device.
//
// For somebody whose pedal is not the HX Stomp. Loosely matched, so "Helix
// LT", "helix lt" and "helix-lt" all reach the same catalog, and a name
// nothing ships for is reported when the catalog is opened, naming the ones
// that do.
//
// WithCatalog wins over this: a catalog somebody generated themselves is a
// stronger statement than the name of a device this binary happens to carry.
func WithDevice(
	name string,
) Option {
	return func(o *options) { o.device = name }
}

// WithStats reads corpus statistics instead of the built-in ones.
func WithStats(
	path string,
) Option {
	return func(o *options) { o.stats = path }
}

// WithSetup reads what the person has, so a build can fit them.
//
// A client option rather than a call's argument, for the reason the Setup is a
// separate document at all: it changes on a different clock. Somebody's bass,
// their pedal and their right hand are the same at the twelfth build as at the
// first, and passing them per call is how the twelfth comes to contradict.
//
// What it changes today is the playing. An ask says how the subject played and
// a Setup says how this person does, and the difference between the two right
// hands is a knob position rather than a surprise at the first rehearsal.
func WithSetup(
	path string,
) Option {
	return func(o *options) { o.setup = path }
}

// WithRigs reads rigs from a directory instead of the ones that ship.
//
// Scaffold and Extend write there too, unless WithUserRigs names somewhere
// else, and need one or the other: a new rig is not written into wherever the
// program happened to run.
func WithRigs(
	dir string,
) Option {
	return func(o *options) { o.rigs = dir }
}

// WithUserRigs layers somebody's own directory of rigs over the ones that
// ship, or over the directory WithRigs named.
//
// A rig of theirs takes the place of one beneath when the two share an
// identifier or alias, in any case, so Rigs lists the rig Rig and Build
// find. Variants are read across both, so a rig of theirs made from a shipped
// one with Extend shows under it. A directory that is not there holds no rigs;
// one that cannot be read is an error.
//
// A file in it that is not a rig is reported by Rigs. It stops Rig,
// Build and Extend only when it may be the rig asked for: when its filename,
// or the id or aliases it states, is the name asked for or a name of the rig
// found. Otherwise one mistake in a directory of their own would stop every
// shipped rig building, and a broken rig of theirs would never be quietly
// passed over for the shipped one it was written to replace.
//
// Scaffold and Extend write here. The library reads no environment for this:
// a program that keeps rigs under $XDG_DATA_HOME resolves that and passes the
// directory in.
func WithUserRigs(
	dir string,
) Option {
	return func(o *options) { o.userRigs = dir }
}

// WithBackupDir is where a slot's old contents go before a device write.
//
// Without it they go to $XDG_STATE_HOME/toneharness/presets, or to
// ~/.local/state/toneharness/presets when that variable is unset.
func WithBackupDir(
	dir string,
) Option {
	return func(o *options) { o.backupDir = dir }
}

// WithCapture receives each device answer a read gets, verbatim.
//
// It is how the wire format was read in the first place: a preset arrives as
// the bytes the device sent, and an answer nothing here decodes arrives as
// JSON.
func WithCapture(
	w io.Writer,
) Option {
	return func(o *options) { o.capture = w }
}

// WithTrace receives every USB frame in and out.
func WithTrace(
	w io.Writer,
) Option {
	return func(o *options) { o.trace = w }
}

// New builds a Client.
//
// Usable with no options at all, because the common case is a caller who
// wants the built-in catalog, the built-in statistics, the rigs that ship and
// whatever device happens to be plugged in. New opens nothing and cannot fail.
func New(
	opts ...Option,
) *Client {
	var o options

	for _, fn := range opts {
		fn(&o)
	}

	if o.devices == nil {
		o.devices = device.NewUSB(o.trace)
	}

	return &Client{opts: o, claim: make(chan struct{}, 1)}
}

// buildFlows builds the slot flows the first time anything asks.
func (c *Client) buildFlows() {
	c.flowsOnce.Do(func() {
		// The flows read the catalog through the Client, so they share the
		// one it opens and keeps.
		c.fileFlows = &fileslots.Flows{Catalogs: c}

		devices := &deviceslots.Flows{Catalogs: c, Capture: c.opts.capture}
		devices.Backups = backup.New(c.opts.backupDir, deviceslots.NewDecoder(devices))
		c.deviceFlows = devices
	})
}

// fileOperations are the flows over a .hls, .hlb or .hlx on disk.
func (c *Client) fileOperations() *fileslots.Flows {
	c.buildFlows()

	return c.fileFlows
}

// deviceOperations are the flows over an attached device.
func (c *Client) deviceOperations() *deviceslots.Flows {
	c.buildFlows()

	return c.deviceFlows
}

// Catalog is the catalog this Client names gear against.
//
// Opened on first use and kept, so a renderer reads the same catalog the
// operation did. A catalog that would not open is not kept, and the next call
// tries again.
func (c *Client) Catalog(
	ctx context.Context,
) (*catalog.Catalog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cat != nil {
		return c.cat, nil
	}

	cat, err := c.openCatalog()
	if err != nil {
		return nil, err
	}

	c.cat = cat

	return cat, nil
}

// openCatalog reads whichever catalog this Client was told to use.
//
// A path somebody gave wins: a catalog they generated themselves is a
// stronger statement than the name of a device this binary happens to carry.
// Then a named device, and then the HX Stomp everything here was written
// against.
func (c *Client) openCatalog() (*catalog.Catalog, error) {
	if c.opts.catalog != "" {
		return catalog.Open(c.opts.catalog)
	}

	if c.opts.device != "" {
		return catalog.ForName(c.opts.device)
	}

	return catalog.BuiltIn()
}

// Devices reports the hardware attached to this machine.
//
// Recognised hardware only: a bus holds keyboards and webcams, and a list of
// those is not an answer to "what can I write a preset to".
func (c *Client) Devices(
	ctx context.Context,
) (Attached, error) {
	return attached.ListWith(ctx, c.opts.devices)
}

// Filter narrows what Blocks reports. All three narrow together.
type Filter struct {
	// Category keeps only blocks of one kind: amp, cab, drive.
	Category string
	// Subcategory keeps only blocks Line 6 tags this way: Guitar, Bass.
	Subcategory string
	// Search keeps only blocks whose name or real-world gear mentions this.
	Search string
}

// Blocks reports what a device can do, narrowed to what was asked for.
func (c *Client) Blocks(
	ctx context.Context,
	f Filter,
) (Blocks, error) {
	cat, err := c.Catalog(ctx)
	if err != nil {
		return Blocks{}, err
	}

	return catalogview.List(cat, catalogview.Filter(f)), nil
}

// Block reports one block and everything it accepts.
func (c *Client) Block(
	ctx context.Context,
	id string,
) (catalog.Block, error) {
	cat, err := c.Catalog(ctx)
	if err != nil {
		return catalog.Block{}, err
	}

	return catalogview.Show(cat, id)
}

// corpus is what every question of the measurements reads.
func (c *Client) corpus() presetsview.Options {
	return presetsview.Options{StatsPath: c.opts.stats, Catalogs: c}
}

// ModelMeasurements reports what players did with one model: how they set
// each parameter across every measured preset that used it.
//
// A model the corpus never saw, including an empty one, is refused rather than
// answered with nothing.
func (c *Client) ModelMeasurements(
	ctx context.Context,
	model string,
) (Measured, error) {
	if err := ctx.Err(); err != nil {
		return Measured{}, err
	}

	return presetsview.Model(ctx, c.corpus(), model)
}

// ChainMeasurements reports what chains tend to hold: which kinds of block,
// and which side of the amp they sit on.
//
// instrument narrows that to guitar or bass. Empty asks about every chain.
func (c *Client) ChainMeasurements(
	ctx context.Context,
	instrument string,
) (Measured, error) {
	if err := ctx.Err(); err != nil {
		return Measured{}, err
	}

	return presetsview.Chains(c.corpus(), instrument)
}

// source is where this Client reads rigs from.
func (c *Client) source() rigs.Source {
	return rigs.Source{Dir: c.opts.rigs, User: c.opts.userRigs}
}

// rigsHome is where Scaffold and Extend write: the directory of somebody's
// own when there is one, and the one WithRigs named otherwise.
func (c *Client) rigsHome() string {
	if c.opts.userRigs != "" {
		return c.opts.userRigs
	}

	return c.opts.rigs
}

// Rigs reads every rig this Client was given.
//
// Dir is the directory WithUserRigs named where there is one, since that
// is where somebody's own rigs are, and the one WithRigs named otherwise.
func (c *Client) Rigs(
	ctx context.Context,
) (Rigs, error) {
	if err := ctx.Err(); err != nil {
		return Rigs{}, err
	}

	return rigs.List(c.source())
}

// Rig reads one rig, and what the rest of the set says about it.
func (c *Client) Rig(
	ctx context.Context,
	id string,
) (Rig, error) {
	if err := ctx.Err(); err != nil {
		return Rig{}, err
	}

	return rigs.Show(c.source(), id)
}

// Tone resolves a request and a setup into the rig they describe.
//
// The step between what somebody wants and what a preset is compiled from.
// Notes come back whether it succeeded or not: a request that could not be
// honoured has usually said why in them, and the error on its own is the half
// that does not help.
func (c *Client) Tone(
	ctx context.Context,
	in Ask,
) (Resolved, error) {
	cat, err := c.openCatalog()
	if err != nil {
		return Resolved{}, err
	}

	lib, err := measured.BuiltIn()
	if err != nil {
		return Resolved{}, err
	}

	return asking.Resolve(ctx, asking.Ask(in), cat, lib)
}

// Backing reads which records back each rig, and holds them to its era.
//
// The rig says which years its gear describes and the manifest says when each
// record was made, and until these were joined nothing noticed that four of
// the shipped rigs derive words from records made on other gear.
func (c *Client) Backing(
	ctx context.Context,
	corpus string,
) ([]Backing, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return rigs.Backing(c.source(), corpus)
}

// NewRig describes a rig to scaffold from the gear it names.
type NewRig struct {
	// ID is the identifier, and the filename stem.
	ID string
	// Name is the player or style, as a person would write it.
	Name string
	// Band is the group, where there is one.
	Band string
	// Instrument is guitar or bass.
	Instrument string
	// Amp is the real-world amplifier. Required: it is the one thing nothing
	// downstream recovers from getting wrong.
	Amp string
	// Cab is the real-world cabinet. Empty takes the amp's own pairing.
	Cab string
	// Pedals are real-world pedals, in signal order.
	Pedals []string
	// Genre is which genres the sound belongs to. Required, because the ask a
	// scaffold writes carries one and a ToneSpec without one is refused.
	Genre []string
}

// Scaffold writes a rig, after checking the gear it names exists.
//
// Checking first is the point. A rig naming gear no device models is only
// found out when somebody tries to build from it, and by then the name has
// usually been copied somewhere else too.
//
// The rig is written into the directory WithUserRigs named, or failing that
// the one WithRigs named. A Client given neither is refused rather than
// writing wherever the program happened to run.
func (c *Client) Scaffold(
	ctx context.Context,
	in NewRig,
) (Scaffolded, error) {
	if err := ctx.Err(); err != nil {
		return Scaffolded{}, err
	}

	return rigs.New(ctx, rigs.NewOptions{
		Dir:        c.rigsHome(),
		ID:         in.ID,
		Name:       in.Name,
		Band:       in.Band,
		Instrument: in.Instrument,
		Amp:        in.Amp,
		Cab:        in.Cab,
		Pedals:     in.Pedals,
		Genre:      in.Genre,
		Catalogs:   c,
	})
}

// ExtendRig describes a rig to start as a copy of another.
type ExtendRig struct {
	// From is the rig to copy, by identifier or alias. Required.
	From string
	// ID is the new rig's identifier, and its filename stem.
	ID string
	// Name is the player or style the copy is about. Empty keeps the name
	// the copied rig has.
	Name string
	// Kind is what the copy is attributed to: artist, band, song, genre or
	// sound. Empty keeps the copied rig's.
	Kind string
}

// Extend writes a new rig as a copy of one that exists.
//
// The copy is a whole rig, comments and citations included, and records the
// rig it came from in `extends`. Nothing merges the two: editing the copy does
// not touch the original. No gear is checked, because the rig it copies
// already resolved when it was written.
//
// The rig is written where Scaffold writes. From is looked for there first,
// then in the rigs beneath: the ones WithRigs named when WithUserRigs
// was given too, and the ones that ship otherwise. The report's Instrument,
// Amp, Cab and Pedals are the copied rig's, and so is its Name unless in.Name
// gave another. in.Name replaces the subject's name and nothing else, encoded
// as YAML so any text is a name.
func (c *Client) Extend(
	ctx context.Context,
	in ExtendRig,
) (Scaffolded, error) {
	if err := ctx.Err(); err != nil {
		return Scaffolded{}, err
	}

	// Without a rig to copy this would scaffold one from no gear at all,
	// which is a different operation with a different check.
	if in.From == "" {
		return Scaffolded{}, fmt.Errorf("%w: name the rig to copy", ErrNoSuchRig)
	}

	// A directory WithRigs named is where the copy goes when there is no
	// directory of their own, so it cannot also be what that one sits on.
	base := ""
	if c.opts.userRigs != "" {
		base = c.opts.rigs
	}

	return rigs.New(ctx, rigs.NewOptions{
		Dir:      c.rigsHome(),
		Base:     base,
		From:     in.From,
		ID:       in.ID,
		Name:     in.Name,
		Kind:     in.Kind,
		Catalogs: c,
	})
}

// Make resolves an ask and the rig beside it into a preset, and writes it to out.
//
// Resolving is what separates this from Compile. An ask carries words, and a
// word moves a control against the chain the compiler built, so the blocks the
// corpus adds and the positions those words reach are decided here. Compile
// lowers a document that has already made those decisions.
//
// Reporting what it chose matters as much as writing the file. A generated
// preset is a set of decisions, and a wrong amp should be visible before
// anybody plugs in rather than after.
//
// existing says what happens to a file already at out: ReplaceExisting puts
// the preset in its place, and KeepExisting refuses it with an error matching
// fs.ErrExist.
func (c *Client) Make(
	ctx context.Context,
	rigID string,
	out string,
	existing Existing,
) (Made, error) {
	return presets.Make(ctx, presets.MakeOptions{
		Deps:       presets.Deps{Catalogs: c},
		RigID:      rigID,
		Source:     c.source(),
		StatsPath:  c.opts.stats,
		SetupPath:  c.opts.setup,
		OutputPath: out,
		Existing:   existing,
	})
}

// MusicPlayers is every player the music corpus names, from the manifests.
//
// The manifests rather than the audio. Measuring a corpus needs the recordings
// and minutes of work per record; this answers what somebody wrote down, which
// is what says whether the records to measure are even there.
func (c *Client) MusicPlayers(
	ctx context.Context,
	corpus string,
) ([]MusicPlayer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	geared, err := c.geared()
	if err != nil {
		return nil, err
	}

	return musicview.Players(os.DirFS(corpus), ".", geared)
}

// MusicBands is every band the corpus names, grouped on the slug so two
// spellings of one band count once.
func (c *Client) MusicBands(
	ctx context.Context,
	corpus string,
) ([]MusicGroup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return musicview.Bands(os.DirFS(corpus), ".")
}

// MusicGenres is every genre the corpus names, and how far each one still is
// from being a distribution worth aiming at.
func (c *Client) MusicGenres(
	ctx context.Context,
	corpus string,
) ([]MusicGroup, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	geared, err := c.geared()
	if err != nil {
		return nil, err
	}

	return musicview.Genres(os.DirFS(corpus), ".", geared)
}

// geared is every rig identifier there is gear for, so a corpus listing can
// say which of its players nothing can be built for.
//
// Read here rather than passed in, because a player having a rig is not a fact
// about the corpus: the corpus is recordings and the rigs are a separate set,
// and a caller should not have to hold both to ask one question.
func (c *Client) geared() (map[string]bool, error) {
	held, err := rigs.List(c.source())
	if err != nil {
		return nil, err
	}

	out := make(map[string]bool, len(held.Rigs))
	for _, k := range held.Rigs {
		out[string(k.Rig.ID)] = true
	}

	return out, nil
}

// MusicRecords is every recording the corpus names, and whether the bass has
// been separated out of it yet.
func (c *Client) MusicRecords(
	ctx context.Context,
	corpus string,
) ([]MusicRecord, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return musicview.Records(os.DirFS(corpus), ".")
}

// MusicGenres measures every genre in a corpus against the players who play
// none of it.
//
// The audio, not the manifests, which is what separates this from
// CorpusGenres: that counts what somebody wrote down and this measures what the
// records sound like. Minutes of work per record, and it needs the recordings on
// disk.
func (c *Client) MeasuredGenres(
	ctx context.Context,
	corpus string,
) ([]audio.Genre, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return audio.GenresMeasured(os.DirFS(corpus), ".")
}

// MeasuredPlayers is what each player's records measure as, and the words that
// earns them against the others.
//
// A word is earned by sitting clear of the other players, so one player alone
// earns nothing: there is nobody to be clear of. Point this at one instrument,
// because a bass centroid sits an octave below a guitar's and a corpus holding
// both would earn every bassist "dark" and mean nothing by it.
//
// Reads the recordings, which costs minutes per record. MusicPlayers answers
// what the corpus holds for the price of a file read.
func (c *Client) MeasuredPlayers(
	ctx context.Context,
	corpus string,
) ([]audio.Player, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return audio.Corpus(os.DirFS(corpus), ".")
}

// MeasuredRecordings is what a directory of recordings measures as, one entry
// per file and the figures they make together.
//
// Separate the instrument out first. A mix measures the band, so a figure taken
// from one describes the arrangement rather than the player.
func (c *Client) MeasuredRecordings(
	ctx context.Context,
	dir string,
) ([]audio.Named, audio.Across, error) {
	if err := ctx.Err(); err != nil {
		return nil, audio.Across{}, err
	}

	got, err := audio.MeasureAll(os.DirFS(dir), ".")
	if err != nil {
		return nil, audio.Across{}, err
	}

	all := make([]audio.Profile, 0, len(got))
	for _, one := range got {
		all = append(all, one.Profile)
	}

	return got, audio.Together(all), nil
}
