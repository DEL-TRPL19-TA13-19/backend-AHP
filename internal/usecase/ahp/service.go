package ahp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"os"
	"ta13-svc/internal/abstraction"
	dto "ta13-svc/internal/dto/ahp"
	"ta13-svc/internal/entity"
	"ta13-svc/internal/factory"
	"ta13-svc/internal/repository"
	"ta13-svc/pkg/response"
	"ta13-svc/pkg/utils/ahp"
	"ta13-svc/pkg/utils/constant"
)

type Service interface {
	FindCriteriaAlternative(ctx context.Context) (*entity.CriteriaData, error)
	FindScoreByCollectionID(ctx context.Context, collectionID *string) ([]entity.AlternativeEntityModel, error)
	FindFinalScoreByCollectionID(ctx context.Context, collectionID *string) ([]entity.AlternativeEntityModel, error)

	UpdateCriteriaAlternative(ctx context.Context, c *entity.CriteriaData) (*entity.CriteriaData, error)

	CalculateAlternativeToPoint(ctx context.Context, collectionID *string) (entity.Matrix, error)
	CalculateScoreAlternativeByCollectionID(ctx context.Context, collectionID *string) ([]entity.ScoreEntityModel, error)
	CalculateFinalScoreByCollectionID(ctx context.Context, collectionID *string) ([]entity.FinalScoreEntityModel, error)
}

type service struct {
	Repository repository.AhpRepository
	Db         *gorm.DB
}

func NewService(f *factory.Factory) *service {
	repository := f.AHPRepository
	db := f.Db
	return &service{repository, db}
}

func (s *service) FindScoreByCollectionID(ctx context.Context, collectionID *string) ([]entity.AlternativeEntityModel, error) {
	datas := make([]entity.AlternativeEntityModel, 0)

	datas, err = s.Repository.FindScoreByCollectionID(ctx, collectionID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datas, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return datas, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	return datas, nil
}

func (s *service) FindFinalScoreByCollectionID(ctx context.Context, collectionID *string) ([]entity.AlternativeEntityModel, error) {
	datas := make([]entity.AlternativeEntityModel, 0)

	datas, err = s.Repository.FindFinalScoreByCollectionID(ctx, collectionID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return datas, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return datas, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	return datas, nil
}

func (s *service) FindCriteriaAlternative(ctx context.Context) (*entity.CriteriaData, error) {
	var result *entity.CriteriaData

	jsonFile, err := os.ReadFile("asset/pairwise.json")
	if err != nil {
		fmt.Println(err)
	}

	var criteriaData entity.CriteriaData
	err = json.Unmarshal(jsonFile, &criteriaData)
	if err != nil {
		fmt.Println(err)
	}

	//TODO:Masih kurang efektif
	var criteriaDataUnedited entity.CriteriaData
	err = json.Unmarshal(jsonFile, &criteriaDataUnedited)
	if err != nil {
		fmt.Println(err)
	}

	//MENCARI SUM DARI MASING MASING COL
	rowsPC := len(criteriaData.PairwiseFromJson)
	colsPC := len(criteriaData.PairwiseFromJson[0])
	colSum := make([]float64, len(criteriaData.PairwiseFromJson))

	for i := 0; i < rowsPC; i++ {
		for j := 0; j < colsPC; j++ {
			colSum[i] += criteriaData.PairwiseFromJson[j][i]
		}
	}

	//NORMALISASI MATRIKS PAIRWISE
	for i := 0; i < rowsPC; i++ {
		for j := 0; j < colsPC; j++ {
			criteriaData.PairwiseFromJson[i][j] /= colSum[j]
		}
	}

	//MENCARI JUMLAH NILAI BARIS DAN KOLOM & MENCARI RATA RATA (BOBOT KRITERIA)
	normalColSum := make([]float64, len(criteriaData.PairwiseFromJson))
	normalRowSum := make([]float64, len(criteriaData.PairwiseFromJson))
	criteriaWeights := make([]float64, len(criteriaData.PairwiseFromJson))

	for i := 0; i < rowsPC; i++ {
		sum := 0.0
		for j := 0; j < colsPC; j++ {
			sum += criteriaData.PairwiseFromJson[i][j]
			normalColSum[i] += criteriaData.PairwiseFromJson[j][i]
			normalRowSum[i] += criteriaData.PairwiseFromJson[i][j]
			criteriaWeights[i] = sum / float64(len(criteriaWeights))
		}
	}

	result = &entity.CriteriaData{
		PairwiseFromJson:        criteriaDataUnedited.PairwiseFromJson,
		PairwiseAfterCalculated: criteriaData.PairwiseFromJson,
		Criteria:                criteriaWeights}

	return result, nil
}

func (s *service) UpdateCriteriaAlternative(ctx context.Context, c *dto.CriteriaAlternativeUpdateRequest) (*entity.CriteriaData, error) {

	jsonFile, err := os.ReadFile("asset/pairwise.json")
	var criteriaData entity.CriteriaData

	err = json.Unmarshal(jsonFile, &criteriaData)

	if err != nil {
		return nil, err
	}

	criteriaData.PairwiseFromJson = c.Pairwise

	b, err := json.Marshal(criteriaData)
	if err != nil {
		return nil, err
	}

	err = os.WriteFile("asset/pairwise.json", b, 0644)
	if err != nil {
		return nil, err
	}

	fmt.Println(string(b))
	result := &entity.CriteriaData{
		PairwiseFromJson: criteriaData.PairwiseFromJson,
	}

	return result, nil
}

func (s *service) CalculateAlternativeToPoint(ctx context.Context, collectionID *string) (entity.Matrix, error) {
	alternatives := make([]entity.AlternativeEntityModel, 0)
	alternatives, err = s.Repository.FindAlternativesByCollectionID(ctx, collectionID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	if len(alternatives) == 0 {
		return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
	}

	matrix := make(entity.Matrix, 0)

	for i := 0; i < len(alternatives); i++ {
		row := entity.Matrix{
			{ahp.TimbulanSampahSubCriteria()[alternatives[i].TimbulanSampah],
				ahp.JarakTPASubCriteria()[alternatives[i].JarakTpa],
				ahp.JarakPemukimanSubCriteria()[alternatives[i].JarakPemukiman],
				ahp.JarakSungaiSubCriteria()[alternatives[i].JarakSungai],
				ahp.PartisipasiMasyarakatSubCriteria()[alternatives[i].PartisipasiMasyarakat],
				ahp.CakupanRumahSubCriteria()[alternatives[i].CakupanRumah],
				ahp.AksesibilitasSubCriteria()[alternatives[i].Aksesibilitas]}}

		matrix = append(matrix, row...)
	}

	return matrix, nil
}

func (s *service) CalculateScoreAlternativeByCollectionID(ctx context.Context, collectionID *string) ([]entity.ScoreEntityModel, error) {
	alternatives := make([]entity.AlternativeEntityModel, 0)
	alternatives, err = s.Repository.FindAlternativesByCollectionID(ctx, collectionID)
	var collection *entity.CollectionEntityModel

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	if len(alternatives) == 0 {
		return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
	}

	checkScores, err := s.Repository.FindScoreByCollectionID(ctx, collectionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	if len(checkScores) > 0 {
		_, err := s.Repository.DeleteAllScoreByCollection(ctx, collectionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
			}
			return nil, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
		}
	}

	matrix := make(entity.Matrix, 0)
	matrix, err = s.CalculateAlternativeToPoint(ctx, collectionID)

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
		}
		return nil, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
	}

	criteriaData, _ := s.FindCriteriaAlternative(ctx)
	criteriaWeights := criteriaData.Criteria

	rowsACS := len(matrix)
	colsACS := len(matrix[0])

	//PERKALIAN MATRIKS ALTERNATIF DENGAN MATRIKS BOBOT
	for i := 0; i < rowsACS; i++ {
		for j := 0; j < colsACS; j++ {
			matrix[i][j] *= criteriaWeights[j]
		}
	}

	scores := make([]entity.ScoreEntityModel, 0)

	for i := 0; i < len(matrix); i++ {
		scores = append(scores, entity.ScoreEntityModel{
			ScoreEntity: entity.ScoreEntity{
				TimbulanSampah:        constant.RoundFloat(matrix[i][0], 3),
				JarakTpa:              constant.RoundFloat(matrix[i][1], 3),
				JarakPemukiman:        constant.RoundFloat(matrix[i][2], 3),
				JarakSungai:           constant.RoundFloat(matrix[i][3], 3),
				PartisipasiMasyarakat: constant.RoundFloat(matrix[i][4], 3),
				CakupanRumah:          constant.RoundFloat(matrix[i][5], 3),
				Aksesibilitas:         constant.RoundFloat(matrix[i][6], 3),
			},
			Entity:        abstraction.Entity{ID: uuid.NewString()},
			CollectionID:  alternatives[i].CollectionID,
			AlternativeID: alternatives[i].ID,
		})
	}

	_, err = s.Repository.CreateScore(ctx, scores)

	collection = &entity.CollectionEntityModel{
		CollectionEntity: entity.CollectionEntity{
			ScoreIsCalculated: true,
		},
	}

	_, err = s.Repository.UpdateCollection(ctx, collectionID, collection)

	if err != nil {
		return nil, err
	}

	return scores, nil

}

func (s *service) CalculateFinalScoreByCollectionID(ctx context.Context, collectionID *string) ([]entity.FinalScoreEntityModel, error) {
	alternativeScores, err := s.CalculateScoreAlternativeByCollectionID(ctx, collectionID)
	var collection *entity.CollectionEntityModel

	if err != nil {
		return nil, err
	}

	if len(alternativeScores) == 0 {
		return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
	}

	checkFinalScores, err := s.Repository.FindFinalScoreByCollectionID(ctx, collectionID)

	if err != nil {
		return nil, err
	}

	if len(checkFinalScores) > 0 {
		_, err := s.Repository.DeleteAllFinalScoreByCollection(ctx, collectionID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return nil, response.ErrorBuilder(&response.ErrorConstant.NotFound, err)
			}
			return nil, response.ErrorBuilder(&response.ErrorConstant.InternalServerError, err)
		}
	}

	finalScores := make([]entity.FinalScoreEntityModel, 0)

	for i := 0; i < len(alternativeScores); i++ {
		finalScores = append(finalScores, entity.FinalScoreEntityModel{
			Entity: abstraction.Entity{ID: uuid.NewString()},
			FinalScoreEntity: entity.FinalScoreEntity{
				FinalScore: (alternativeScores[i].TimbulanSampah + alternativeScores[i].JarakTpa + alternativeScores[i].JarakPemukiman + alternativeScores[i].JarakSungai + alternativeScores[i].PartisipasiMasyarakat + alternativeScores[i].CakupanRumah + alternativeScores[i].Aksesibilitas) * 100,
				Rank:       0,
			},
			AlternativeID: alternativeScores[i].AlternativeID,
			CollectionID:  alternativeScores[i].CollectionID,
		})
	}

	_, err = s.Repository.CreateFinalScore(ctx, finalScores)

	collection = &entity.CollectionEntityModel{
		CollectionEntity: entity.CollectionEntity{
			FinalScoreIsCalculated: true,
		},
	}

	_, err = s.Repository.UpdateCollection(ctx, collectionID, collection)

	if err != nil {
		return nil, err
	}

	return finalScores, nil
}
