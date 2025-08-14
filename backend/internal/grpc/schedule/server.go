// Package schedule реализует gRPC сервер для работы с расписанием
package schedule

import (
	"context"
	"log"

	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/jwt"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
	pb "github.com/Ultrahd-dev/student-schedule-app/backend/proto/gen/schedule"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Server реализует gRPC сервис для работы с расписанием
type Server struct {
	pb.UnimplementedScheduleServiceServer
	scheduleService *schedule.Service
	jwtManager      *jwt.Manager
	userService     *users.Service
}

// NewServer создает новый gRPC сервер для расписания
func NewServer(scheduleService *schedule.Service, jwtManager *jwt.Manager, userService *users.Service) *Server {
	return &Server{
		scheduleService: scheduleService,
		jwtManager:      jwtManager,
		userService:     userService,
	}
}

// GetScheduleForGroup получает расписание для группы на определенную дату
func (s *Server) GetScheduleForGroup(ctx context.Context, req *pb.GetScheduleForGroupRequest) (*pb.GetScheduleForGroupResponse, error) {
	log.Printf("Получен запрос на получение расписания для группы: %s", req.GroupName)

	// Проверяем токен
	claims, err := s.jwtManager.ParseToken(req.Token)
	if err != nil {
		log.Printf("Ошибка проверки токена: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Неверный токен")
	}

	// Проверяем существование пользователя (временно не используем данные)
	_, err = s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		log.Printf("Ошибка получения пользователя %s: %v", claims.UserID, err)
		return nil, status.Errorf(codes.NotFound, "Пользователь не найден")
	}

	// TODO: Проверить права доступа пользователя к расписанию группы
	// Например, студент может просматривать только расписание своей группы

	// Получаем расписание для группы
	scheduleEntries, err := s.scheduleService.GetScheduleForGroup(ctx, req.GroupName, req.Date.AsTime())
	if err != nil {
		log.Printf("Ошибка получения расписания для группы %s: %v", req.GroupName, err)
		return nil, status.Errorf(codes.Internal, "Ошибка получения расписания: %v", err)
	}

	// Преобразуем записи расписания в формат protobuf
	var pbSchedule []*pb.ScheduleEntry
	for _, entry := range scheduleEntries {
		// Преобразуем SourceType в protobuf enum
		var sourceTypeEnum pb.ScheduleSourceType
		switch entry.SourceType {
		case "main":
			sourceTypeEnum = pb.ScheduleSourceType_SCHEDULE_SOURCE_TYPE_MAIN
		case "change":
			sourceTypeEnum = pb.ScheduleSourceType_SCHEDULE_SOURCE_TYPE_CHANGE
		default:
			// По умолчанию используем UNDEFINED или логируем ошибку
			sourceTypeEnum = pb.ScheduleSourceType_SCHEDULE_SOURCE_TYPE_UNSPECIFIED
			log.Printf("Неизвестный тип источника: %s", entry.SourceType)
		}

		pbEntry := &pb.ScheduleEntry{
			Id:         entry.ID.String(),
			GroupName:  entry.GroupName,
			Date:       timestamppb.New(entry.Date),
			TimeStart:  entry.TimeStart,
			TimeEnd:    entry.TimeEnd,
			Subject:    entry.Subject,
			Teacher:    entry.Teacher,
			Classroom:  entry.Classroom,
			SourceType: sourceTypeEnum,
			SourceId:   entry.SourceID.String(),
		}
		pbSchedule = append(pbSchedule, pbEntry)
	}

	// Формируем ответ
	response := &pb.GetScheduleForGroupResponse{
		Success:  true,
		Message:  "Расписание получено успешно",
		Schedule: pbSchedule,
	}

	log.Printf("Расписание для группы %s на дату %s успешно получено", req.GroupName, req.Date.AsTime().Format("2006-01-02"))
	return response, nil
}

// GetActiveScheduleSnapshot получает активный снапшот расписания
func (s *Server) GetActiveScheduleSnapshot(ctx context.Context, req *pb.GetActiveScheduleSnapshotRequest) (*pb.GetActiveScheduleSnapshotResponse, error) {
	log.Println("Получен запрос на получение активного снапшота расписания")

	// Проверяем токен
	claims, err := s.jwtManager.ParseToken(req.Token)
	if err != nil {
		log.Printf("Ошибка проверки токена: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Неверный токен")
	}

	// Получаем информацию о пользователе (временно не используем данные)
	_, err = s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		log.Printf("Ошибка получения пользователя %s: %v", claims.UserID, err)
		return nil, status.Errorf(codes.NotFound, "Пользователь не найден")
	}

	// TODO: Проверить права доступа пользователя

	// Получаем активный снапшот
	snapshot, err := s.scheduleService.GetActiveScheduleSnapshot(ctx)
	if err != nil {
		log.Printf("Ошибка получения активного снапшота: %v", err)
		return nil, status.Errorf(codes.Internal, "Ошибка получения снапшота: %v", err)
	}

	// Преобразуем снапшот в формат protobuf
	pbSnapshot := &pb.ScheduleSnapshot{
		Id:          snapshot.ID.String(),
		Name:        snapshot.Name,
		PeriodStart: timestamppb.New(snapshot.PeriodStart),
		PeriodEnd:   timestamppb.New(snapshot.PeriodEnd),
		Data:        string(snapshot.Data),
		CreatedAt:   timestamppb.New(snapshot.CreatedAt),
		SourceUrl:   snapshot.SourceURL,
		IsActive:    snapshot.IsActive,
	}

	// Формируем ответ
	response := &pb.GetActiveScheduleSnapshotResponse{
		Success:  true,
		Message:  "Активный снапшот получен успешно",
		Snapshot: pbSnapshot,
	}

	log.Println("Активный снапшот расписания успешно получен")
	return response, nil
}

// GetScheduleSnapshotsHistory получает историю снапшотов расписания
func (s *Server) GetScheduleSnapshotsHistory(ctx context.Context, req *pb.GetScheduleSnapshotsHistoryRequest) (*pb.GetScheduleSnapshotsHistoryResponse, error) {
	log.Println("Получен запрос на получение истории снапшотов расписания")

	// Проверяем токен
	claims, err := s.jwtManager.ParseToken(req.Token)
	if err != nil {
		log.Printf("Ошибка проверки токена: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Неверный токен")
	}

	// Получаем информацию о пользователе (временно не используем данные)
	_, err = s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		log.Printf("Ошибка получения пользователя %s: %v", claims.UserID, err)
		return nil, status.Errorf(codes.NotFound, "Пользователь не найден")
	}

	// TODO: Проверить права доступа пользователя (администратор может видеть все)

	// TODO: Реализовать получение истории снапшотов
	// Пока возвращаем пустой список
	pbSnapshots := []*pb.ScheduleSnapshot{}

	// Формируем ответ
	response := &pb.GetScheduleSnapshotsHistoryResponse{
		Success:   true,
		Message:   "История снапшотов получена успешно",
		Snapshots: pbSnapshots,
	}

	log.Println("История снапшотов расписания успешно получена")
	return response, nil
}

// RegisterService регистрирует сервис в gRPC сервере
func RegisterService(grpcServer *grpc.Server, scheduleService *schedule.Service, jwtManager *jwt.Manager, userService *users.Service) {
	pb.RegisterScheduleServiceServer(grpcServer, NewServer(scheduleService, jwtManager, userService))
}

