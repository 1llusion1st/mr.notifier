package notifier

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/sirupsen/logrus"
	"strings"
)

type Command interface {
	Execute(c *Config, bot *tgbotapi.BotAPI, admin *AdminHandler, sourceMsgId int) error
}

type CommandStart struct{}

func (cmd *CommandStart) Execute(c *Config, bot *tgbotapi.BotAPI, admin *AdminHandler, sourceMsgId int) error {
	reply := tgbotapi.NewMessage(c.Telegram.AdminChatId, "wellcome to MR.notifier bot!")
	reply.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Projects"),
			tgbotapi.NewKeyboardButton("Reviewers"),
		),
	)
	_, err := bot.Send(reply)
	return err
}

type CommandListProjects struct{}

func (cmd *CommandListProjects) Execute(c *Config, bot *tgbotapi.BotAPI, admin *AdminHandler, sourceMsgId int) error {
	logrus.Debugf("CommandListProjects Execute called with %d", sourceMsgId)
	projects := c.ListProjects()
	reviewers := c.ListReviewers()
	logrus.Debugf("projects: %v reviewers: %v", projects, reviewers)
	for _, project := range projects {
		logrus.Debugf("processing project %s", project)

		markup := NewInlineMarkUp(3)
		//markup.AddButton(admin.NewCallbackButton("REMOVE PRJ ?", nil))
		//markup.AddRow()

		for _, reviewer := range reviewers {
			if c.HasProjectReviewer(project, reviewer) {
				markup.AddButton(
					admin.NewCallbackButton(
						fmt.Sprintf("- %s", reviewer),
						&CommandRemoveProjectReviewer{
							Reviewer:          reviewer,
							Project:           project,
							OnSuccessCallback: &CommandListProjects{},
						}))
			} else {
				markup.AddButton(
					admin.NewCallbackButton(
						fmt.Sprintf("+ %s", reviewer),
						&CommandAddProjectReviewer{
							Reviewer:          reviewer,
							Project:           project,
							OnSuccessCallback: &CommandListProjects{},
						}))
			}
		}

		msg := tgbotapi.NewMessage(c.Telegram.AdminChatId, project)
		msg.ReplyMarkup = tgbotapi.InlineKeyboardMarkup{
			InlineKeyboard: markup.markup,
		}
		var err error
		logrus.Debugf("sourceMsgId: %d", sourceMsgId)
		if sourceMsgId == 0 {
		} else {
			logrus.Debugf("deleting %d:%d", c.Telegram.AdminChatId, sourceMsgId)
			_, err := bot.DeleteMessage(tgbotapi.NewDeleteMessage(c.Telegram.AdminChatId, sourceMsgId))
			if err != nil {
				logrus.Errorf("can't delete message(%d:%d): %v", c.Telegram.AdminChatId, sourceMsgId, err)
			}
			logrus.Debugf("deleted ?")
		}
		logrus.Debugf("sending response msg")
		_, err = bot.Send(msg)
		if err != nil {
			logrus.Errorf("can't send message: %v", err)
		}
	}
	return nil
}

type CommandListReviewers struct {
	Project string
}

func (cmd *CommandListReviewers) Execute(c *Config, bot *tgbotapi.BotAPI, admin *AdminHandler, sourceMsgId int) error {
	reviewers := c.ListReviewers()
	msg := tgbotapi.NewMessage(c.Telegram.AdminChatId, fmt.Sprintf("Reviewers:\n%s", strings.Join(reviewers, "\n")))
	_, err := bot.Send(msg)
	return err
}

type CommandAddProjectReviewer struct {
	Project           string
	Reviewer          string
	OnSuccessCallback Command
}

func (cmd *CommandAddProjectReviewer) Execute(c *Config, bot *tgbotapi.BotAPI, admin *AdminHandler, sourceMsgId int) error {
	logrus.Debugf("CommandAddProjectReviewer.Execute called")
	added := c.AddReviewerToProject(cmd.Project, cmd.Reviewer)
	logrus.Debugf("Added %s to %s ? %v", cmd.Reviewer, cmd.Project, added)
	logrus.Debugf("checking cmd.OnSuccessCallback: %v", cmd.OnSuccessCallback)
	if cmd.OnSuccessCallback != nil {
		logrus.Debugf("calling callback-2")
		return cmd.OnSuccessCallback.Execute(c, bot, admin, sourceMsgId)
	}
	return nil
}

type CommandRemoveProjectReviewer struct {
	Project           string
	Reviewer          string
	OnSuccessCallback Command
}

func (cmd *CommandRemoveProjectReviewer) Execute(c *Config, bot *tgbotapi.BotAPI, admin *AdminHandler, sourceMsgId int) error {
	logrus.Debugf("CommandRemoveProjectReviewer.Execute called")
	removed := c.RemoveReviewerFromProject(cmd.Project, cmd.Reviewer)
	logrus.Debugf("Removed %s from %s ? %v", cmd.Reviewer, cmd.Project, removed)
	logrus.Debugf("checking cmd.OnSuccessCallback: %v", cmd.OnSuccessCallback)
	if cmd.OnSuccessCallback != nil {
		logrus.Debugf("calling callback-2")
		return cmd.OnSuccessCallback.Execute(c, bot, admin, sourceMsgId)
	}
	return nil
}
