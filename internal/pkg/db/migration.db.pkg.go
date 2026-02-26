package database

import (
	"fmt"
	"go-boilerplate/internal/common/models"
	"go-boilerplate/internal/pkg/logger"
)

func (db *Database) RunMigrations() error {
	logger.Info.Println("Starting database migrations...")

	// Create extensions first
	// if err := db.createExtensions(); err != nil {
	// 	return fmt.Errorf("failed to create extensions: %w", err)
	// }

	// Note: analisis_status is now handled as VARCHAR type in models
	// No custom types needed

	// Define models in dependency order
	models := []interface{}{
		&models.Transaction{},
	}

	for _, model := range models {
		logger.Info.Printf("Migrating model: %T", model)
		if err := db.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	// Create indexes after all tables are created
	// if err := db.createIndexes(); err != nil {
	// 	return fmt.Errorf("failed to create indexes: %w", err)
	// }

	// Create triggers for updated_at columns
	// if err := db.createTriggers(); err != nil {
	// 	return fmt.Errorf("failed to create triggers: %w", err)
	// }

	logger.Info.Println("Database migrations completed successfully")
	return nil
}

func (db *Database) createExtensions() error {
	query := `CREATE EXTENSION IF NOT EXISTS "pgcrypto";`
	return db.Exec(query).Error
}

func (db *Database) createIndexes() error {
	indexes := []string{
		// Customer indexes
		`CREATE INDEX IF NOT EXISTS idx_customers_team_id ON customers(team_id);`,
		`CREATE INDEX IF NOT EXISTS idx_customers_nik ON customers(nik);`,
		`CREATE INDEX IF NOT EXISTS idx_customers_status ON customers(status);`,
		`CREATE INDEX IF NOT EXISTS idx_customers_created_at ON customers(created_at);`,
		`CREATE INDEX IF NOT EXISTS idx_customers_deleted_at ON customers(deleted_at);`,
		`CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_code_team ON customers(code, team_id);`,

		// Credit applications indexes
		`CREATE INDEX IF NOT EXISTS idx_credit_applications_customer_id ON credit_applications(customer_id);`,
		`CREATE INDEX IF NOT EXISTS idx_credit_applications_code ON credit_applications(code);`,
		`CREATE INDEX IF NOT EXISTS idx_credit_applications_threshold_value ON credit_applications(threshold_value);`,
		`CREATE INDEX IF NOT EXISTS idx_credit_applications_status ON credit_applications(status);`,

		// Data room indexes
		`CREATE INDEX IF NOT EXISTS idx_data_room_team_id ON data_room(team_id);`,
		`CREATE INDEX IF NOT EXISTS idx_data_room_status ON data_room(status);`,
		`CREATE INDEX IF NOT EXISTS idx_dataroom_name_team ON data_room(name, team_id);`,

		// Bank statement docs indexes
		`CREATE INDEX IF NOT EXISTS idx_bank_statement_docs_team_id ON bank_statement_docs(team_id);`,
		`CREATE INDEX IF NOT EXISTS idx_bank_statement_docs_status ON bank_statement_docs(status);`,
		`CREATE INDEX IF NOT EXISTS idx_bank_statement_docs_account_number ON bank_statement_docs(account_number);`,
		`CREATE INDEX IF NOT EXISTS idx_bank_statement_docs_period ON bank_statement_docs(period);`,

		// Bank statement indexes
		`CREATE INDEX IF NOT EXISTS idx_bank_statement_docs_id ON bank_statement(bank_statement_docs_id);`,

		// Junction table indexes
		`CREATE INDEX IF NOT EXISTS idx_dataroom_bankstatement_data_room_id ON data_room_bank_statement_docs(data_room_id);`,
		`CREATE INDEX IF NOT EXISTS idx_dataroom_collateral_data_room_id ON data_room_collateral_docs(data_room_id);`,
		`CREATE INDEX IF NOT EXISTS idx_dataroom_slikojk_data_room_id ON data_room_slik_ojk_docs(data_room_id);`,
		`CREATE INDEX IF NOT EXISTS idx_dataroom_relatedparties_data_room_id ON data_room_related_parties(data_room_id);`,
	}

	for _, query := range indexes {
		if err := db.Exec(query).Error; err != nil {
			logger.Error.Printf("Error creating index: %s, Error: %v", query, err)
			return err
		}
	}

	return nil
}

func (db *Database) createTriggers() error {
	// Create the trigger function first
	triggerFunction := `
	CREATE OR REPLACE FUNCTION update_updated_at_column()
	RETURNS TRIGGER AS $$
	BEGIN
		NEW.updated_at = NOW();
		RETURN NEW;
	END;
	$$ language 'plpgsql';`

	if err := db.Exec(triggerFunction).Error; err != nil {
		return err
	}

	// Tables that need updated_at triggers
	tables := []string{
		"customers",
		"credit_applications",
		"data_room",
		"bank_statement_docs",
		"bank_statement_indicator_categories",
		"bank_statement_indicator_types",
		"bank_statement_analyses",
		"collateral_docs",
		"slik_ojk_docs",
		"slik_ojk_doc_analysis",
		"related_parties_masters",
		"related_parties_lists",
		"related_parties",
	}

	for _, table := range tables {
		triggerQuery := fmt.Sprintf(`
		DROP TRIGGER IF EXISTS update_%s_updated_at ON %s;
		CREATE TRIGGER update_%s_updated_at
		BEFORE UPDATE ON %s
		FOR EACH ROW EXECUTE PROCEDURE update_updated_at_column();`,
			table, table, table, table)

		if err := db.Exec(triggerQuery).Error; err != nil {
			logger.Error.Printf("Error creating trigger for table %s: %v", table, err)
			return err
		}
	}

	return nil
}
