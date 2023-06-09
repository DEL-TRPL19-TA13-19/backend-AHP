package alternative

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"ta13-svc/internal/abstraction"
	dto "ta13-svc/internal/dto/alternative"
	"ta13-svc/internal/entity"
	"ta13-svc/internal/factory"
	"ta13-svc/internal/repository"
	"ta13-svc/pkg/response"
	"ta13-svc/pkg/utils/trxmanager"
)

type Service interface {
	FindAll(ctx context.Context) ([]entity.AlternativeEntityModel, error)
	FindByID(ctx context.Context, payload *dto.AlternativeGetByIDRequest) (*dto.AlternativeGetByIDResponse, error)
	FindByCollectionID(ctx context.Context, payload *dto.AlternativeGetByCollectionIDRequest) ([]entity.AlternativeEntityModel, error)
	Create(ctx context.Context, payload *dto.AlternativeCreateRequest) (*dto.AlternativeCreateResponse, error)
	Update(ctx context.Context, payload *dto.AlternativeUpdateRequest) (*dto.AlternativeUpdateResponse, error)
	Delete(ctx context.Context, payload *dto.AlternativeDeleteRequest) (*dto.AlternativeDeleteResponse, error)
}

type service struct {
	Repository repository.AlternativeRepository
	Db         *gorm.DB
}

func NewService(f *factory.Factory) *service {
	repository := f.AlternativeRepository
	db := f.Db
	return &service{repository, db}
}

func (s *service) FindAll(ctx context.Context) ([]entity.AlternativeEntityModel, error) {
	datas := make([]entity.AlternativeEntityModel, 0)

	datas, err = s.Repository.FindAll(ctx)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datas, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return datas, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	return datas, nil
}

func (s *service) FindByID(ctx context.Context, payload *dto.AlternativeGetByIDRequest) (*dto.AlternativeGetByIDResponse, error) {
	var result *dto.AlternativeGetByIDResponse

	data, err := s.Repository.FindByID(ctx, &payload.ID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return result, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return result, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	result = &dto.AlternativeGetByIDResponse{
		Datas: *data,
	}

	return result, nil
}

func (s *service) FindByCollectionID(ctx context.Context, payload *dto.AlternativeGetByCollectionIDRequest) ([]entity.AlternativeEntityModel, error) {
	datas := make([]entity.AlternativeEntityModel, 0)

	datas, err = s.Repository.FindByCollectionID(ctx, &payload.CollectionID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datas, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return datas, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	return datas, nil
}

func (s *service) Create(ctx context.Context, payload *dto.AlternativeCreateRequest) (*dto.AlternativeCreateResponse, error) {
	var result *dto.AlternativeCreateResponse
	var data *entity.AlternativeEntityModel

	if err = trxmanager.New(s.Db).WithTrxV2(ctx, func(ctx context.Context, f *factory.Factory) error {
		alternativeRepository := f.AlternativeRepository

		data = &entity.AlternativeEntityModel{
			Entity:            abstraction.Entity{ID: uuid.NewString()},
			AlternativeEntity: payload.AlternativeEntity,
			CollectionID:      payload.CollectionID,
		}

		_, err := alternativeRepository.FindByCollectionID(ctx, &payload.CollectionID)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}

		_, err = alternativeRepository.Create(ctx, data)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}

		return nil
	}); err != nil {
		return result, err

	}

	result = &dto.AlternativeCreateResponse{
		AlternativeEntityModel: *data,
	}

	return result, nil
}

func (s *service) Update(ctx context.Context, payload *dto.AlternativeUpdateRequest) (*dto.AlternativeUpdateResponse, error) {
	var result *dto.AlternativeUpdateResponse
	var data *entity.AlternativeEntityModel

	if err = trxmanager.New(s.Db).WithTrxV2(ctx, func(ctx context.Context, f *factory.Factory) error {
		alternativeRepository := f.AlternativeRepository

		data = &entity.AlternativeEntityModel{
			AlternativeEntity: payload.AlternativeEntity,
			Entity:            abstraction.Entity{ID: payload.ID},
		}

		_, err := alternativeRepository.FindByID(ctx, &payload.ID)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}

		_, err = alternativeRepository.Update(ctx, &payload.ID, data)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		return nil
	}); err != nil {
		return result, err
	}

	result = &dto.AlternativeUpdateResponse{
		AlternativeEntityModel: *data,
	}

	return result, nil
}

func (s *service) Delete(ctx context.Context, payload *dto.AlternativeDeleteRequest) (*dto.AlternativeDeleteResponse, error) {
	var result *dto.AlternativeDeleteResponse
	var data *entity.AlternativeEntityModel

	if err = trxmanager.New(s.Db).WithTrxV2(ctx, func(ctx context.Context, f *factory.Factory) error {
		alternativeRepository := f.AlternativeRepository
		data = &entity.AlternativeEntityModel{
			AlternativeEntity: payload.AlternativeEntity,
		}
		_, err := alternativeRepository.FindByID(ctx, &payload.ID)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}

		_, err = alternativeRepository.Delete(ctx, &payload.ID, data)
		if err != nil {
			return response.ErrorBuilder(&response.ErrorConstant.UnprocessableEntity, err)
		}
		return nil
	}); err != nil {
		return result, err
	}

	result = &dto.AlternativeDeleteResponse{
		ID: &payload.ID,
	}

	return result, nil
}
