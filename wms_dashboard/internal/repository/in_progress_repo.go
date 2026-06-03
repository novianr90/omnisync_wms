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
		query = query.Where("doc_type LIKE ?", docType+"%")
	}
	if search != "" {
		query = query.Where("document_no LIKE ?", "%"+search+"%")
	}

	err := query.Order("created_at DESC").Find(&docs).Error
	return docs, err
}
