package manage

import (
	"gorm.io/gorm"

	"yingyan.local/backend/internal/asset"
	"yingyan.local/backend/internal/credit"
	"yingyan.local/backend/internal/generation"
	"yingyan.local/backend/internal/redemption"
	"yingyan.local/backend/internal/retouch"
)

type Service struct {
	db           *gorm.DB
	credits      *credit.Service
	redemptions  *redemption.Service
	assets       *asset.Service
	generations  *generation.Service
	retouches    *retouch.Service
	bcryptCost   int
	publicWebURL string
}

func New(
	db *gorm.DB,
	credits *credit.Service,
	redemptions *redemption.Service,
	assets *asset.Service,
	generations *generation.Service,
	retouches *retouch.Service,
	bcryptCost int,
	publicWebURL string,
) *Service {
	return &Service{
		db: db, credits: credits, redemptions: redemptions, assets: assets,
		generations: generations, retouches: retouches,
		bcryptCost:   bcryptCost,
		publicWebURL: publicWebURL,
	}
}

func pageValues(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}
