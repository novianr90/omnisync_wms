package repository

import (
	"wms_dashboard/internal/database"
	"wms_dashboard/internal/models"
)

// FetchInProgressDocs queries the in_progress_documents SQL view with dynamic filters
func FetchInProgressDocs(docType string, search string) ([]models.InProgressDoc, error) {
	var docs []models.InProgressDoc
	query := database.DB.Model(&models.InProgressDoc{})

	if docType != "" && docType != "All" {
		query = query.Where("LOWER(doc_type) LIKE LOWER(?)", docType+"%")
	}
	if search != "" {
		query = query.Where("LOWER(document_no) LIKE LOWER(?)", "%"+search+"%")
	}

	err := query.Order("created_at DESC").Find(&docs).Error
	return docs, err
}
