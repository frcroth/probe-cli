package ooni

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/jmoiron/sqlx"
	"github.com/ooni/probe-cli/config"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/internal/legacy"
	"github.com/ooni/probe-cli/utils"
	"github.com/pkg/errors"
)

// Onboarding process
func Onboarding(c *config.Config) error {
	log.Info("Onboarding starting")

	// To prevent races we always must acquire the config file lock before
	// changing it.
	c.Lock()
	c.InformedConsent = true
	c.Unlock()

	if err := c.Write(); err != nil {
		log.Warnf("Failed to save informed consent: %v", err)
		return err
	}
	return nil
}

// Context for OONI Probe
type Context struct {
	Config   *config.Config
	DB       *sqlx.DB
	Location *utils.LocationInfo

	Home    string
	TempDir string

	dbPath     string
	configPath string
}

// MaybeLocationLookup will lookup the location of the user unless it's already cached
func (c *Context) MaybeLocationLookup() error {
	if c.Location == nil {
		return c.LocationLookup()
	}
	return nil
}

// LocationLookup lookup the location of the user via geoip
func (c *Context) LocationLookup() error {
	var err error

	geoipDir := utils.GeoIPDir(c.Home)

	c.Location, err = utils.GeoIPLookup(geoipDir)
	if err != nil {
		return err
	}

	return nil
}

// MaybeOnboarding will run the onboarding process only if the informed consent
// config option is set to false
func (c *Context) MaybeOnboarding() error {
	if c.Config.InformedConsent == false {
		if err := Onboarding(c.Config); err != nil {
			return errors.Wrap(err, "onboarding")
		}
	}
	return nil
}

// Init the OONI manager
func (c *Context) Init() error {
	var err error

	if err = legacy.MaybeMigrateHome(); err != nil {
		return errors.Wrap(err, "migrating home")
	}

	if err = MaybeInitializeHome(c.Home); err != nil {
		return err
	}

	if c.configPath != "" {
		log.Debugf("Reading config file from %s", c.configPath)
		c.Config, err = config.ReadConfig(c.configPath)
	} else {
		log.Debug("Reading default config file")
		c.Config, err = ReadDefaultConfigPaths(c.Home)
	}
	if err != nil {
		return err
	}

	c.dbPath = utils.DBDir(c.Home, "main")
	log.Debugf("Connecting to database sqlite3://%s", c.dbPath)
	db, err := database.Connect(c.dbPath)
	if err != nil {
		return err
	}
	c.DB = db

	tempDir, err := ioutil.TempDir("", "ooni")
	if err != nil {
		return errors.Wrap(err, "creating TempDir")
	}
	c.TempDir = tempDir

	return nil
}

// NewContext instance.
func NewContext(configPath string, homePath string) *Context {
	return &Context{
		Home:       homePath,
		Config:     &config.Config{},
		configPath: configPath,
	}
}

// MaybeInitializeHome does the setup for a new OONI Home
func MaybeInitializeHome(home string) error {
	firstRun := false
	for _, d := range utils.RequiredDirs(home) {
		if _, e := os.Stat(d); e != nil {
			firstRun = true
			if err := os.MkdirAll(d, 0700); err != nil {
				return err
			}
		}
	}
	if firstRun == true {
		log.Info("This is the first time you are running OONI Probe. Downloading some files.")
		geoipDir := utils.GeoIPDir(home)
		log.Debugf("Downloading GeoIP database files")
		if err := utils.DownloadGeoIPDatabaseFiles(geoipDir); err != nil {
			return err
		}
		log.Debugf("Downloading legacy GeoIP database Files")
		if err := utils.DownloadLegacyGeoIPDatabaseFiles(geoipDir); err != nil {
			return err
		}
	}

	return nil
}

// ReadDefaultConfigPaths from common locations.
func ReadDefaultConfigPaths(home string) (*config.Config, error) {
	var paths = []string{
		filepath.Join(home, "config.json"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			c, err := config.ReadConfig(path)
			if err != nil {
				return nil, err
			}
			return c, nil
		}
	}

	// Run from the default config
	return config.ReadConfig(paths[0])
}
