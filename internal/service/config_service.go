package service

import (
	"context"
	"log"

	"github.com/alias-asso/iosu/internal/database"
	"github.com/alias-asso/iosu/internal/repository"
)

type ConfigService struct {
	repo repository.ConfigRepository
}

func NewConfigService(repo repository.ConfigRepository) ConfigService {
	return ConfigService{
		repo: repo,
	}
}

func (s *ConfigService) CreateDefaultConfig(ctx context.Context) {
	config := &database.Config{
		Singleton:     1,
		SiteTitle:     "title",
		MainText:      "example main text",
		SecondaryText: "example secondary text",
		HelpContent:   "default *help* content (markdown rendered)",
		RulesContent:  "default *rules* content (markdown rendered)",
	}
	ok, err := s.repo.CreateIfNotExist(ctx, config)
	if err != nil {
		log.Fatalln("Error creating default site config")
	}

	if ok {
		log.Println("Filling default site config in database with example values. Please change it with the iosu binary.")
	}
}

type UpdateConfigInput struct {
	SiteTitle      *string
	MainText       *string
	SecondaryText  *string
	CurrentContest *string
	HelpContent    *string
	RulesContent   *string
}

func (s *ConfigService) UpdateConfig(ctx context.Context, input UpdateConfigInput) error {

	update := database.Config{}

	if input.SiteTitle != nil {
		update.SiteTitle = *input.SiteTitle
	}

	if input.MainText != nil {
		update.MainText = *input.MainText
	}

	if input.SecondaryText != nil {
		update.SecondaryText = *input.SecondaryText
	}

	if input.CurrentContest != nil {
		update.CurrentContest = *input.CurrentContest
	}

	if input.HelpContent != nil {
		update.HelpContent = *input.HelpContent
	}

	if input.RulesContent != nil {
		update.RulesContent = *input.RulesContent
	}

	return s.repo.Update(ctx, update)
}

func (s *ConfigService) GetConfig(ctx context.Context) (database.Config, error) {
	config, err := s.repo.Get(ctx)
	if err != nil {
		return database.Config{}, repository.ErrConfigNotFound
	}
	return config, nil
}
