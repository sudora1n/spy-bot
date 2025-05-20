package files

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"ssuspy-bot/config"
	"ssuspy-bot/consts"
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

	var (
		fileLimit uint64
		isLocal   bool
	)

	if config.Config.TelegramBot.ApiURL == "" {
		fileLimit = consts.MAX_FILE_SIZE_BYTES
	} else {
		fileLimit, isLocal = consts.MAX_FILE_SIZE_BYTES_LOCAL, true
	}

	if job.File.FileSize > int64(fileLimit) {
		_, err = w.bot.SendMessage(ctx, tu.Message(
			tu.ID(job.UserID),
			job.Loc.MustLocalize(&i18n.LocalizeConfig{
				MessageID: "errors.errorFileTooBig",
				TemplateData: map[string]any{
					"FileSize":  job.File.FileSize,
					"FileLimit": humanize.Bytes(fileLimit),
				},
			}),
		).
			WithReplyParameters(&telego.ReplyParameters{
				MessageID: job.MessageID,
				ChatID:    tu.ID(job.UserID),
			}))
		return err
	}

	fileExists, err := w.service.CreateFileIfNotExists(ctx, job.File.FileID, job.UserID, job.ChatID)
	if err != nil || !fileExists {
		return err
	}

	fileNetPath, err := w.bot.GetFile(ctx, &telego.GetFileParams{FileID: job.File.FileID})
	if err != nil {
		return err
	}

	var (
		file   telego.InputFile
		closer io.Closer
	)

	if isLocal {
		file, closer, err = w.createInputMediaLocal(fileNetPath)
	} else {
		file, closer, err = w.createInputMedia(fileNetPath)
	}
	if err != nil {
		return err
	}
	defer closer.Close()

	inputMedia := utils.CreateInputMediaFromFileInfoByFile(file, job.File.Type, job.Caption)

	return utils.SendMediaInGroups(w.bot, ctx, job.UserID, []telego.InputMedia{inputMedia}, job.MessageID)
}

func (w Worker) createInputMedia(file *telego.File) (result telego.InputFile, closer io.Closer, err error) {
	url := w.bot.FileDownloadURL(file.FilePath)
	resp, err := http.Get(url)
	if err != nil {
		return result, nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	result = tu.FileFromReader(
		resp.Body,
		fmt.Sprintf("file.%s", filepath.Ext(file.FilePath)),
	)

	return result, resp.Body, nil
}

func (w Worker) createInputMediaLocal(file *telego.File) (result telego.InputFile, closer io.Closer, err error) {
	f, err := os.Open(file.FilePath)
	if err != nil {
		return result, nil, fmt.Errorf("open local file failed: %v", err)
	}

	result = tu.File(f)
	return result, f, nil
}
