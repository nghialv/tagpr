package tagpr

import (
	"os"
	"strconv"

	"github.com/Songmu/gitconfig"
	"github.com/google/go-github/v47/github"
)

const (
	defaultConfigFile    = ".tagpr"
	defaultConfigContent = `# config file for the tagpr in git config format
# The tagpr generates the initial configuration, which you can rewrite to suit your environment.
# CONFIGURATIONS:
#   tagpr.releaseBranch
#       Generally, it is "main." It is the branch for releases. The pcpr tracks this branch,
#       creates or updates a pull request as a release candidate, or tags when they are merged.
#
#   tagpr.versionFile
#       Versioning file containing the semantic version needed to be updated at release.
#       It will be synchronized with the "git tag".
#       Often this is a meta-information file such as gemspec, setup.cfg, package.json, etc.
#       Sometimes the source code file, such as version.go or Bar.pm, is used.
#       If you do not want to use versioning files but only git tags, specify the "-" string here.
#       You can specify multiple version files by comma separated strings.
#
#   tagpr.vPrefix
#       Flag whether or not v-prefix is added to semver when git tagging. (e.g. v1.2.3 if true)
#       This is only a tagging convention, not how it is described in the version file.
#
#   tagpr.command (Optional)
#       Command to change files just before release.
#
#   tagpr.tmplate (Optional)
#       Pull request template in go template format
[tagpr]
`
	envReleaseBranch    = "TAGPR_RELEASE_BRANCH"
	envVersionFile      = "TAGPR_VERSION_FILE"
	envVPrefix          = "TAGPR_VPREFIX"
	envCommand          = "TAGPR_COMMAND"
	envTemplate         = "TAGPR_TEMPLATE"
	configReleaseBranch = "tagpr.releaseBranch"
	configVersionFile   = "tagpr.versionFile"
	configVPrefix       = "tagpr.vPrefix"
	configCommand       = "tagpr.command"
	configTemplate      = "tagpr.template"
)

type config struct {
	releaseBranch *configValue
	versionFile   *configValue
	command       *configValue
	template      *configValue
	vPrefix       *bool

	conf      string
	gitconfig *gitconfig.Config
}

func newConfig(gitPath string) (*config, error) {
	cfg := &config{
		conf:      defaultConfigFile,
		gitconfig: &gitconfig.Config{GitPath: gitPath, File: defaultConfigFile},
	}
	err := cfg.Reload()
	return cfg, err
}

func (cfg *config) Reload() error {
	if rb := os.Getenv(envReleaseBranch); rb != "" {
		cfg.releaseBranch = &configValue{
			value:  rb,
			source: srcEnv,
		}
	} else {
		out, err := cfg.gitconfig.Get(configReleaseBranch)
		if err == nil {
			cfg.releaseBranch = &configValue{
				value:  out,
				source: srcConfigFile,
			}
		}
	}

	if rb := os.Getenv(envVersionFile); rb != "" {
		cfg.versionFile = &configValue{
			value:  rb,
			source: srcEnv,
		}
	} else {
		out, err := cfg.gitconfig.Get(configVersionFile)
		if err == nil {
			cfg.versionFile = &configValue{
				value:  out,
				source: srcConfigFile,
			}
		}
	}

	if vPrefix := os.Getenv(envVPrefix); vPrefix != "" {
		b, err := strconv.ParseBool(vPrefix)
		if err != nil {
			return err
		}
		cfg.vPrefix = github.Bool(b)
	} else {
		b, err := cfg.gitconfig.Bool(configVPrefix)
		if err == nil {
			cfg.vPrefix = github.Bool(b)
		}
	}

	if command := os.Getenv(envCommand); command != "" {
		cfg.command = &configValue{
			value:  command,
			source: srcEnv,
		}
	} else {
		command, err := cfg.gitconfig.Get(configCommand)
		if err == nil {
			cfg.command = &configValue{
				value:  command,
				source: srcConfigFile,
			}
		}
	}

	if tmpl := os.Getenv(envTemplate); tmpl != "" {
		cfg.template = &configValue{
			value:  tmpl,
			source: srcEnv,
		}
	} else {
		template, err := cfg.gitconfig.Get(configTemplate)
		if err == nil {
			cfg.template = &configValue{
				value:  template,
				source: srcConfigFile,
			}
		}
	}

	return nil
}

func (cfg *config) set(key, value string) error {
	if !exists(cfg.conf) {
		if err := cfg.initializeFile(); err != nil {
			return err
		}
	}
	if value == "" {
		value = "-" // value "-" represents null (really?)
	}
	_, err := cfg.gitconfig.Do(key, value)
	if err != nil {
		// in this case, config file might be invalid or broken, so retry once.
		if err = cfg.initializeFile(); err != nil {
			return err
		}
		_, err = cfg.gitconfig.Do(key, value)
	}
	return err
}

func (cfg *config) initializeFile() error {
	if err := os.RemoveAll(cfg.conf); err != nil {
		return err
	}
	if err := os.WriteFile(cfg.conf, []byte(defaultConfigContent), 0666); err != nil {
		return err
	}
	return nil
}

func (cfg *config) SetRelaseBranch(br string) error {
	if err := cfg.set(configReleaseBranch, br); err != nil {
		return err
	}
	cfg.releaseBranch = &configValue{
		value:  br,
		source: srcDetect,
	}
	return nil
}

func (cfg *config) SetVersionFile(fpath string) error {
	if err := cfg.set(configVersionFile, fpath); err != nil {
		return err
	}
	cfg.versionFile = &configValue{
		value:  fpath,
		source: srcDetect,
	}
	return nil
}

func (cfg *config) SetVPrefix(vPrefix bool) error {
	if err := cfg.set(configVPrefix, strconv.FormatBool(vPrefix)); err != nil {
		return err
	}
	cfg.vPrefix = github.Bool(vPrefix)
	return nil
}

func (cfg *config) ReleaseBranch() *configValue {
	return cfg.releaseBranch
}

func (cfg *config) VersionFile() *configValue {
	return cfg.versionFile
}

func (cfg *config) Command() *configValue {
	return cfg.command
}

func (cfg *config) Template() *configValue {
	return cfg.template
}

type configValue struct {
	value  string
	source configSource
}

func (cv *configValue) String() string {
	if cv.value == "-" {
		return ""
	}
	return cv.value
}

func (cv *configValue) Empty() bool {
	return cv.String() == ""
}

type configSource int

const (
	srcEnv configSource = iota
	srcConfigFile
	srcDetect
)
