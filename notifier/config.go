package notifier

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"strings"
	"sync"
)

type ProjectInfo struct {
	Project   string   `arg:"" name:"project"`
	Reviewers []string `arg:"" name:"reviewer"`
}

func (p *ProjectInfo) HasReviewer(reviewer string) bool {
	for _, reviewerPresent := range p.Reviewers {
		if reviewerPresent == reviewer {
			return true
		}
	}
	return false
}

func (c *ProjectInfo) AddReviewer(reviewer string) {
	c.Reviewers = append(c.Reviewers, reviewer)
}

func (p *ProjectInfo) RemoveReviewer(reviewerToRemove string) {
	newReviewers := make([]string, 0)
	for _, reviewer := range p.Reviewers {
		if reviewer != reviewerToRemove {
			newReviewers = append(newReviewers, reviewer)
		}
	}
	p.Reviewers = newReviewers
}

type Config struct {
	Telegram struct {
		BotApi        string `arg:"" name:"bot-api" yaml:"bot-api"`
		ChannelChatId int64  `arg:"" name:"channel-id" yaml:"channel-chat-id"`
		ThreadId      int64  `arg:"" name:"thread-id" yaml:"thread-id"`
		AdminChatId   int64  `arg:"" name:"admin-id" yaml:"admin-chat-id"`
	} `embed:"" prefix:"telegram."`
	Projects    []ProjectInfo `kong:"-"`
	Reviewers   []string      `kong:"-"`
	WebHookPath string        `arg:"" name:"web-hook-path" yaml:"web-hook-path"`
	WebHookPort int           `arg:"" name:"webhook-port" yaml:"web-hook-port"`
	//GitToken    string        `arg:"" name:"git-token" yaml:"git-token"`
	AddProjects []string   `yaml:"-" arg:"" name:"new-projects" help:"list or projects to handle: project,reviewer1,reviewer2"`
	lock        sync.Mutex `kong:"-" yaml:"-"`
	changed     bool
	syncPath    string
}

func (c *Config) FastLock() func() {
	c.lock.Lock()
	logrus.Debugf("locked")
	return func() {
		if c.changed && c.syncPath != "" {
			// sync conf
			data, err := yaml.Marshal(c)
			if err != nil {
				logrus.Errorf("can't marshal config: %v", err)
			} else {
				err = ioutil.WriteFile(c.syncPath, data, 0644)
				if err != nil {
					logrus.Errorf("can't synchronize config: %v", err)
				}
			}

			c.changed = false
		}

		c.lock.Unlock()
		logrus.Debugf("unlocked")
	}
}

func (c *Config) markChanged() {
	c.changed = true
}

func (c *Config) setSyncPath(path string) {
	c.syncPath = path
}

func (c *Config) ParseProjectsToAdd() error {
	defer (c.FastLock())()

	projectsInfo := make([]ProjectInfo, len(c.AddProjects))
	reviewers := make(map[string]struct{})
	for idx := range c.AddProjects {
		if len(c.AddProjects[idx]) == 0 {
			continue
		}
		splited := strings.Split(c.AddProjects[idx], ",")
		projectsInfo[idx] = ProjectInfo{
			Project: splited[0],
		}
		if len(splited) > 1 {
			projectsInfo[idx].Reviewers = splited[1:]
			for _, reviewer := range splited[1:] {
				reviewers[reviewer] = struct{}{}
			}
		}
	}
	c.Projects = projectsInfo
	c.Reviewers = make([]string, 0)
	for reviewer := range reviewers {
		c.Reviewers = append(c.Reviewers, reviewer)
	}
	return nil
}

func (c *Config) GetProjectReviewers(project string) []string {
	defer (c.FastLock())()
	for _, prj := range c.Projects {
		if prj.Project == project {
			return prj.Reviewers
		}
	}
	return nil
}

func (c *Config) hasProject(project string) bool {
	for _, prj := range c.Projects {
		if prj.Project == project {
			return true
		}
	}
	return false
}

func (c *Config) IsAdmin(chatId int64) bool {
	return c.Telegram.AdminChatId == chatId
}

func (c *Config) IsChannel(chatId int64) bool {
	return c.Telegram.ChannelChatId == chatId
}

func (c *Config) UnexpectedChat(chatId int64) bool {
	return !(c.IsChannel(chatId) || c.IsAdmin(chatId))
}

func (c *Config) ListProjects() []string {
	defer (c.FastLock())()
	projects := make([]string, len(c.Projects))
	for idx := range c.Projects {
		projects[idx] = c.Projects[idx].Project
	}
	return projects
}

func (c *Config) ListReviewers() []string {
	return c.Reviewers
}

func (c *Config) hasReviewer(reviewerToFind string) bool {
	logrus.Debugf("all reviewers: %v need fount: %s", c.Reviewers, reviewerToFind)
	for _, reviewer := range c.Reviewers {
		if reviewer == reviewerToFind {
			return true
		}
	}
	return false
}

func (c *Config) AddReviewer(reviewer string) bool {
	defer (c.FastLock())()
	if c.hasReviewer(reviewer) {
		return false
	}
	c.Reviewers = append(c.Reviewers, reviewer)
	c.markChanged()
	return true
}

func (c *Config) RemoveReviewer(reviewerToRemove string) bool {
	defer (c.FastLock())()
	if c.hasReviewer(reviewerToRemove) {
		oldReviewers := c.Reviewers
		c.Reviewers = make([]string, 0)
		for _, reviewer := range oldReviewers {
			if reviewerToRemove == reviewer {
				continue
			}
			c.Reviewers = append(c.Reviewers, reviewer)
		}
		for idx := range c.Projects {
			if c.Projects[idx].HasReviewer(reviewerToRemove) {
				c.Projects[idx].RemoveReviewer(reviewerToRemove)
			}
		}
		c.markChanged()
		return true
	}
	return false
}

func (c *Config) HasProjectReviewer(project string, reviewer string) bool {
	reviewers := c.GetProjectReviewers(project)

	defer (c.FastLock())()
	if c.hasProject(project) {
		for _, prjReviewer := range reviewers {
			if prjReviewer == reviewer {
				return true
			}
		}
	}
	return false
}

func (c *Config) AddReviewerToProject(project string, reviewer string) bool {
	defer (c.FastLock())()
	if c.hasProject(project) {
		for idx := range c.Projects {
			if c.Projects[idx].Project == project {
				if c.Projects[idx].HasReviewer(reviewer) {
					return false
				}
				c.Projects[idx].AddReviewer(reviewer)
				c.markChanged()
				return true
			}
		}
	}
	return false
}

func (c *Config) RemoveReviewerFromProject(project string, reviewer string) bool {
	defer (c.FastLock())()
	if c.hasReviewer(reviewer) {
		for idx := range c.Projects {
			if c.Projects[idx].Project == project {
				logrus.Debugf("project found")
				if !c.Projects[idx].HasReviewer(reviewer) {
					logrus.Debugf("no reviewer found")
					return false
				}
				logrus.Debugf("reviewer found")
				c.Projects[idx].RemoveReviewer(reviewer)
				c.markChanged()
				return true
			}
		}
	}
	logrus.Debugf("reviewer not found in list")
	return false
}
