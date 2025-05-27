package files

import (
	"context"
	"fmt"
	"os"
	"ssuspy-bot/consts"
	"ssuspy-bot/format"
	"ssuspy-bot/locales"
	"ssuspy-bot/redis"
	"ssuspy-bot/repository"
	"ssuspy-bot/utils"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mymmrac/telego"
	tu "github.com/mymmrac/telego/telegoutil"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog/log"
)

type Worker struct {
	service *repository.MongoRepository
	rdb     *redis.Redis
	bot     *telego.Bot
}

func NewWorker(
	service *repository.MongoRepository,
	rdb *redis.Redis,
	bot *telego.Bot,
) *Worker {
	return &Worker{
		service: service,
		rdb:     rdb,
		bot:     bot,
	}
}

func (w Worker) Work(ctx context.Context, workerI int) {
	log.Info().Int("workerID", workerI).Msg("Worker started")

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

	if job.File.FileSize > consts.MAX_FILE_SIZE_BYTES {
		_, err = w.bot.SendMessage(ctx, tu.Message(
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

	fileNetPath, err := w.bot.GetFile(ctx, &telego.GetFileParams{FileID: job.File.FileID})
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

	return utils.SendMediaInGroups(w.bot, ctx, job.UserID, []telego.InputMedia{inputMedia}, job.MessageID)
}
