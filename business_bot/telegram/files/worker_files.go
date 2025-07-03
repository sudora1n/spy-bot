package files

import (
	"context"
	"fmt"
	"os"
	"ssuspy-bot/consts"
	"ssuspy-bot/manager"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/telegram/format"
	"ssuspy-bot/telegram/locales"
	"ssuspy-bot/utils"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog/log"
)

type Worker struct {
	service    *repository.MongoRepository
	rdb        *redis.Redis
	botManager *manager.BotManager
}

func NewWorker(
	service *repository.MongoRepository,
	rdb *redis.Redis,
	botManager *manager.BotManager,
) *Worker {
	return &Worker{
		service:    service,
		rdb:        rdb,
		botManager: botManager,
	}
}

func (w Worker) Work(ctx context.Context) {
	for {
		res, err := w.rdb.DequeueJob(ctx, consts.REDIS_QUEUE_FILES, 5*time.Second)
		if err != nil {
			log.Printf("Ошибка при чтении из Redis: %v", err)
			time.Sleep(time.Second)
			continue
		}
		if res == nil {
			continue
		}

		if err := w.process(res); err != nil {
			log.Warn().Err(err).Int64("userID", res.UserID).Msg("failed send protected files")
		}
	}
}

func (w Worker) process(job *redis.Job) (err error) {
	ctx := context.TODO()
	loc := locales.NewLocalizer(job.UserLanguageCode)

	bot, ok := w.botManager.GetBot(job.BotID)
	if !ok {
		log.Error().Int64("botID", job.BotID).Msg("no bot found")
		return fmt.Errorf("no bot found")
	}

	if job.File.FileSize > consts.MAX_FILE_SIZE_BYTES {
		_, err = bot.Bot.SendMessage(ctx, tu.Message(
			tu.ID(job.UserID),
			loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "errors.errorFileTooBig",
				TemplateData: map[string]any{
					"FileSize":  job.File.FileSize,
					"FileLimit": humanize.Bytes(consts.MAX_FILE_SIZE_BYTES),
				},
			}),
		).
			WithReplyParameters(&telego.ReplyParameters{
				MessageID: job.MessageID,
				ChatID:    tu.ID(job.UserID),
			}))
		return err
	}

	fileNetPath, err := bot.Bot.GetFile(ctx, &telego.GetFileParams{FileID: job.File.FileID})
	if err != nil {
		return err
	}

	f, err := os.Open(fileNetPath.FilePath)
	if err != nil {
		return fmt.Errorf("open local file failed: %v", err)
	}
	defer f.Close()

	var caption string
	if job.Caption != "" {
		caption = loc.MustLocalize(&i18n.LocalizeConfig{
			MessageID: "sendMediaInGroups",
			TemplateData: map[string]string{
				"Result": format.Caption(job.Caption),
			},
		})
	}

	file := tu.File(f)
	inputMedia := utils.CreateInputMediaFromFileInfoByFile(file, job.File.Type, caption)

	return utils.SendMediaInGroups(bot.Bot, ctx, job.UserID, []telego.InputMedia{inputMedia}, job.MessageID)
}
