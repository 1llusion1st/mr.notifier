package main

import (
	"encoding/json"
	"fmt"
	"github.com/1llusion1st/mr.notifier/notifier"
	"github.com/alecthomas/kong"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type CmdGenerateConfig struct {
	OutFile string `arg:"" name:"config-file"`
	notifier.Config
}

func (c *CmdGenerateConfig) Run() error {
	err := c.ParseProjectsToAdd()
	if err != nil {
		return err
	}
	//if c.Telegram.AdminChatId > 0 {
	//	c.Telegram.AdminChatId = -c.Telegram.AdminChatId
	//}
	if c.Telegram.ChannelChatId > 0 {
		c.Telegram.ChannelChatId = -c.Telegram.ChannelChatId
	}
	data, err := yaml.Marshal(c.Config)
	if err != nil {
		return err
	}
	file := c.OutFile
	if c.OutFile == "-" {
		file = os.Stdout.Name()
	}
	return ioutil.WriteFile(file, data, 0644)
}

type CmdRunMRNotifier struct {
	ConfigFile      string `arg:"" name:"config-file"`
	notifier.Config `kong:"-"`
}

func (c *CmdRunMRNotifier) Run() error {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.Infof("reading config file %s ...", c.ConfigFile)
	sourceFile, err := ioutil.ReadFile(c.ConfigFile)
	if err != nil {
		return err
	}
	logrus.Infof("parsing config ...")
	err = yaml.Unmarshal(sourceFile, &c.Config)
	if err != nil {
		return err
	}
	logrus.Debugf("config: %v", c)

	logrus.Infof("preparing tg.bot...")
	bot, err := tgbotapi.NewBotAPI(c.Telegram.BotApi)
	if err != nil {
		return err
	}
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		return err
	}
	logrus.Debugf("updates chan: %v", updates)
	admin := notifier.NewAdminHandler(c.ConfigFile, bot, &c.Config)
	go admin.HandleUpdates(updates)

	logrus.Infof("preparing http handler...")
	http.HandleFunc(c.WebHookPath, func(w http.ResponseWriter, r *http.Request) {
		logrus.Debugf("[%s] new request...", r.Method)
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		logrus.Debugf("data received")
		var header RequestHeader
		err = json.Unmarshal(data, &header)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		logrus.Debugf("unmarshaled header: %s", header.EventType)
		if header.EventType != "merge_request" {
			w.WriteHeader(http.StatusOK)
			return
		}

		var request MergeRequestOpened
		err = json.Unmarshal(data, &request)
		if err != nil {
			logrus.Errorf("bad MR request: %v", err)
			w.WriteHeader(http.StatusExpectationFailed)
			return
		}
		if request.ObjectAttributes.State == "opened" {
			// MR was opened
			project := request.Project.WebURL
			author := request.User.Email + ": " + request.User.Username + ": " + request.User.Name
			sourceBranch := request.ObjectAttributes.SourceBranch
			targetBranch := request.ObjectAttributes.TargetBranch
			reviewers := c.GetProjectReviewers(project)
			tittle := request.ObjectAttributes.Title
			description := request.ObjectAttributes.Description
			url := request.ObjectAttributes.URL

			reviewersLinks := ""
			if len(reviewers) > 0 {
				reviewersLinks = strings.Join(reviewers, ", ")
			}

			_, err := bot.Send(tgbotapi.NewMessage(c.Telegram.ChannelChatId, fmt.Sprintf(`#MR
			project: %s
			by: %s
			route: %s -> %s
			link: %s
			info: %s
			description: %s
			reviwers: %s
			`,
				project,
				author,
				sourceBranch, targetBranch,
				url,
				tittle,
				description,
				reviewersLinks,
			)))
			if err != nil {
				logrus.Errorf("can't send message: %v", err)
			}
		}

	})
	logrus.Infof("starting http server ...")
	_ = http.ListenAndServe(fmt.Sprintf(":%d", c.WebHookPort), nil)
	return nil
}

var cli struct {
	Generate CmdGenerateConfig `cmd:""`
	Run      CmdRunMRNotifier  `cmd:""`
}

func main() {
	ctx := kong.Parse(&cli)
	// Call the Run() method of the selected parsed command.
	err := ctx.Run()
	ctx.FatalIfErrorf(err)
}

type RequestHeader struct {
	ObjectKind string `json:"object_kind"`
	EventType  string `json:"event_type"`
}

type MergeRequestOpened struct {
	ObjectKind string `json:"object_kind"`
	EventType  string `json:"event_type"`
	User       struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
		Email     string `json:"email"`
	} `json:"user"`
	Project struct {
		ID                int         `json:"id"`
		Name              string      `json:"name"`
		Description       string      `json:"description"`
		WebURL            string      `json:"web_url"`
		AvatarURL         interface{} `json:"avatar_url"`
		GitSSHURL         string      `json:"git_ssh_url"`
		GitHTTPURL        string      `json:"git_http_url"`
		Namespace         string      `json:"namespace"`
		VisibilityLevel   int         `json:"visibility_level"`
		PathWithNamespace string      `json:"path_with_namespace"`
		DefaultBranch     string      `json:"default_branch"`
		Homepage          string      `json:"homepage"`
		URL               string      `json:"url"`
		SSHURL            string      `json:"ssh_url"`
		HTTPURL           string      `json:"http_url"`
	} `json:"project"`
	Repository struct {
		Name        string `json:"name"`
		URL         string `json:"url"`
		Description string `json:"description"`
		Homepage    string `json:"homepage"`
	} `json:"repository"`
	ObjectAttributes struct {
		ID              int    `json:"id"`
		Iid             int    `json:"iid"`
		TargetBranch    string `json:"target_branch"`
		SourceBranch    string `json:"source_branch"`
		SourceProjectID int    `json:"source_project_id"`
		AuthorID        int    `json:"author_id"`
		AssigneeIds     []int  `json:"assignee_ids"`
		AssigneeID      int    `json:"assignee_id"`
		ReviewerIds     []int  `json:"reviewer_ids"`
		Title           string `json:"title"`
		//CreatedAt                   time.Time   `json:"created_at"`
		//UpdatedAt                   time.Time   `json:"updated_at"`
		MilestoneID                 interface{} `json:"milestone_id"`
		State                       string      `json:"state"`
		BlockingDiscussionsResolved bool        `json:"blocking_discussions_resolved"`
		WorkInProgress              bool        `json:"work_in_progress"`
		FirstContribution           bool        `json:"first_contribution"`
		MergeStatus                 string      `json:"merge_status"`
		TargetProjectID             int         `json:"target_project_id"`
		Description                 string      `json:"description"`
		URL                         string      `json:"url"`
		Source                      struct {
			Name              string      `json:"name"`
			Description       string      `json:"description"`
			WebURL            string      `json:"web_url"`
			AvatarURL         interface{} `json:"avatar_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			GitHTTPURL        string      `json:"git_http_url"`
			Namespace         string      `json:"namespace"`
			VisibilityLevel   int         `json:"visibility_level"`
			PathWithNamespace string      `json:"path_with_namespace"`
			DefaultBranch     string      `json:"default_branch"`
			Homepage          string      `json:"homepage"`
			URL               string      `json:"url"`
			SSHURL            string      `json:"ssh_url"`
			HTTPURL           string      `json:"http_url"`
		} `json:"source"`
		Target struct {
			Name              string      `json:"name"`
			Description       string      `json:"description"`
			WebURL            string      `json:"web_url"`
			AvatarURL         interface{} `json:"avatar_url"`
			GitSSHURL         string      `json:"git_ssh_url"`
			GitHTTPURL        string      `json:"git_http_url"`
			Namespace         string      `json:"namespace"`
			VisibilityLevel   int         `json:"visibility_level"`
			PathWithNamespace string      `json:"path_with_namespace"`
			DefaultBranch     string      `json:"default_branch"`
			Homepage          string      `json:"homepage"`
			URL               string      `json:"url"`
			SSHURL            string      `json:"ssh_url"`
			HTTPURL           string      `json:"http_url"`
		} `json:"target"`
		LastCommit struct {
			ID        string    `json:"id"`
			Message   string    `json:"message"`
			Timestamp time.Time `json:"timestamp"`
			URL       string    `json:"url"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		} `json:"last_commit"`
		Labels []struct {
			ID          int       `json:"id"`
			Title       string    `json:"title"`
			Color       string    `json:"color"`
			ProjectID   int       `json:"project_id"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
			Template    bool      `json:"template"`
			Description string    `json:"description"`
			Type        string    `json:"type"`
			GroupID     int       `json:"group_id"`
		} `json:"labels"`
		Action              string `json:"action"`
		DetailedMergeStatus string `json:"detailed_merge_status"`
	} `json:"object_attributes"`
	Labels []struct {
		ID          int       `json:"id"`
		Title       string    `json:"title"`
		Color       string    `json:"color"`
		ProjectID   int       `json:"project_id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
		Template    bool      `json:"template"`
		Description string    `json:"description"`
		Type        string    `json:"type"`
		GroupID     int       `json:"group_id"`
	} `json:"labels"`
	Changes struct {
		UpdatedByID struct {
			Previous interface{} `json:"previous"`
			Current  int         `json:"current"`
		} `json:"updated_by_id"`
		UpdatedAt struct {
			Previous string `json:"previous"`
			Current  string `json:"current"`
		} `json:"updated_at"`
		Labels struct {
			Previous []struct {
				ID          int       `json:"id"`
				Title       string    `json:"title"`
				Color       string    `json:"color"`
				ProjectID   int       `json:"project_id"`
				CreatedAt   time.Time `json:"created_at"`
				UpdatedAt   time.Time `json:"updated_at"`
				Template    bool      `json:"template"`
				Description string    `json:"description"`
				Type        string    `json:"type"`
				GroupID     int       `json:"group_id"`
			} `json:"previous"`
			Current []struct {
				ID          int       `json:"id"`
				Title       string    `json:"title"`
				Color       string    `json:"color"`
				ProjectID   int       `json:"project_id"`
				CreatedAt   time.Time `json:"created_at"`
				UpdatedAt   time.Time `json:"updated_at"`
				Template    bool      `json:"template"`
				Description string    `json:"description"`
				Type        string    `json:"type"`
				GroupID     int       `json:"group_id"`
			} `json:"current"`
		} `json:"labels"`
	} `json:"changes"`
	Assignees []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	} `json:"assignees"`
	Reviewers []struct {
		ID        int    `json:"id"`
		Name      string `json:"name"`
		Username  string `json:"username"`
		AvatarURL string `json:"avatar_url"`
	} `json:"reviewers"`
}
