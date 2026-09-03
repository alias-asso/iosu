package app

import (
	"context"
	"database/sql"
	"errors"

	"github.com/alias-asso/iosu/internal/store/sqlc"
)

// SiteConfig returns the editable site-wide text.
func (a *App) SiteConfig(ctx context.Context) (SiteConfig, error) {
	c, err := a.store.GetSiteConfig(ctx)
	if errors.Is(err, sql.ErrNoRows) {
		return SiteConfig{}, nil
	}
	return c, err
}

// EnsureSiteConfig writes the placeholder site config the first time the
// database is created.
func (a *App) EnsureSiteConfig(ctx context.Context) (created bool, err error) {
	n, err := a.store.CreateSiteConfigIfMissing(ctx, sqlc.CreateSiteConfigIfMissingParams{
		SiteTitle:      "iosu",
		MainText:       "Texte principal à modifier depuis la ligne de commande.",
		SecondaryText:  "Texte secondaire à modifier depuis la ligne de commande.",
		HelpContent:    "Page d'aide *par défaut* (rendue depuis du markdown).",
		RulesContent:   "Page de règles *par défaut* (rendue depuis du markdown).",
		LegalContent:   "Mentions légales *par défaut* (rendues depuis du markdown).",
		CreditsContent: "Crédits *par défaut* (rendus depuis du markdown).",
	})
	return n > 0, err
}

// UpdateSiteConfig applies the non-nil fields of in.
func (a *App) UpdateSiteConfig(ctx context.Context, in sqlc.UpdateSiteConfigParams) error {
	// A slug naming no contest would leave the home page pointing at a 404.
	// The empty string is allowed: it is how the active contest is turned off.
	if in.CurrentContest.Valid && in.CurrentContest.String != "" {
		if _, err := a.Contest(ctx, in.CurrentContest.String); err != nil {
			return err
		}
	}
	return a.store.UpdateSiteConfig(ctx, in)
}
