package models

import (
	"errors"

	"gorm.io/gorm"
)

// ways to get a smtp config
// 1. if they pass in a smtp config id, we get that one
// 2. if they don't pass in a smtp config id, we get the default one
// 3. if they pass in a smtp config id, but it's not found, we return an error
// 4. if they don't pass in a smtp config id, but the default one is not found, we return an error
// 5. they can also pass the provider name, and we get that provider's config

func GetSMTPConfig(teamID string, smtpConfigID string, providerName string, db *gorm.DB) (*SMTPConfig, error) {

	if smtpConfigID == "" && providerName == "" {

		// get the default smtp config for the team
		smtpConfig := &SMTPConfig{}
		if err := db.Where("team_id = ? AND is_default = true", teamID).First(smtpConfig).Error; err != nil {
			return nil, err
		}
		return smtpConfig, nil
	}

	if smtpConfigID != "" {
		smtpConfig := &SMTPConfig{}
		if err := db.Where("id = ? AND team_id = ?", smtpConfigID, teamID).First(smtpConfig).Error; err != nil {
			return nil, err
		}
		return smtpConfig, nil
	}

	if providerName != "" {
		smtpConfig := &SMTPConfig{}
		if err := db.Where("provider = ? AND team_id = ?", providerName, teamID).First(smtpConfig).Error; err != nil {
			return nil, err
		}
		return smtpConfig, nil
	}

	return nil, errors.New("no smtp config found")
}

// GetTeamByName retrieves a team from the database by its name
func GetTeamByName(name string, db *gorm.DB) (*Team, error) {
	team := &Team{}
	if err := db.Where("name = ?", name).First(team).Error; err != nil {
		return nil, err
	}
	return team, nil
}
