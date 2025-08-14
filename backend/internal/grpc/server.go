// Package grpc реализует gRPC сервер для API Gateway
// В соответствии с требованием ТЗ: "API Gateway - gRPC API"
package grpc

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	schedulegrpc "github.com/Ultrahd-dev/student-schedule-app/backend/internal/grpc/schedule" // Для регистрации
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/jwt"
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/schedule" // Пакет schedule
	"github.com/Ultrahd-dev/student-schedule-app/backend/internal/users"
	pb "github.com/Ultrahd-dev/student-schedule-app/backend/proto/gen/users"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

// Server реализует gRPC сервис для работы с пользователями
type Server struct {
	pb.UnimplementedUserServiceServer
	userService *users.Service
	jwtManager  *jwt.Manager
}

// NewServer создает новый gRPC сервер
func NewServer(userService *users.Service, jwtManager *jwt.Manager) *Server {
	return &Server{
		userService: userService,
		jwtManager:  jwtManager,
	}
}

// RegisterStudent регистрирует нового студента
func (s *Server) RegisterStudent(ctx context.Context, req *pb.RegisterStudentRequest) (*pb.RegisterResponse, error) {
	log.Printf("Получен запрос на регистрацию студента: %s", req.Email)

	// Подготавливаем данные для регистрации
	input := users.RegisterStudentInput{
		RegisterUserInput: users.RegisterUserInput{
			Email:    req.Email,
			Password: req.Password,
			Role:     users.RoleStudent,
		},
		GroupName:     req.GroupName,
		Faculty:       req.Faculty,
		Course:        int(req.Course),
		StudentNumber: req.StudentNumber,
	}

	// Регистрируем студента
	user, student, err := s.userService.RegisterStudent(ctx, input)
	if err != nil {
		log.Printf("Ошибка регистрации студента %s: %v", req.Email, err)
		return nil, status.Errorf(codes.Internal, "Ошибка регистрации: %v", err)
	}

	// Формируем ответ
	response := &pb.RegisterResponse{
		Success: true,
		Message: "Студент успешно зарегистрирован",
		User: &pb.User{
			Id:        user.ID.String(),
			Email:     user.Email,
			Role:      pb.UserRole(pb.UserRole_value[string(user.Role)]),
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			IsActive:  user.IsActive,
		},
		Profile: &pb.RegisterResponse_StudentProfile{
			StudentProfile: &pb.StudentProfile{
				UserId:        student.UserID.String(),
				GroupName:     student.GroupName,
				Faculty:       student.Faculty,
				Course:        int32(student.Course),
				StudentNumber: student.StudentNumber,
			},
		},
	}

	log.Printf("Студент %s успешно зарегистрирован", req.Email)
	return response, nil
}

// RegisterTeacher регистрирует нового преподавателя
func (s *Server) RegisterTeacher(ctx context.Context, req *pb.RegisterTeacherRequest) (*pb.RegisterResponse, error) {
	log.Printf("Получен запрос на регистрацию преподавателя: %s", req.Email)

	// Подготавливаем данные для регистрации
	input := users.RegisterTeacherInput{
		RegisterUserInput: users.RegisterUserInput{
			Email:    req.Email,
			Password: req.Password,
			Role:     users.RoleTeacher,
		},
		FullName:   req.FullName,
		Department: req.Department,
		Position:   req.Position,
		TeacherID:  req.TeacherId,
	}

	// Регистрируем преподавателя
	user, teacher, err := s.userService.RegisterTeacher(ctx, input)
	if err != nil {
		log.Printf("Ошибка регистрации преподавателя %s: %v", req.Email, err)
		return nil, status.Errorf(codes.Internal, "Ошибка регистрации: %v", err)
	}

	// Формируем ответ
	response := &pb.RegisterResponse{
		Success: true,
		Message: "Преподаватель успешно зарегистрирован",
		User: &pb.User{
			Id:        user.ID.String(),
			Email:     user.Email,
			Role:      pb.UserRole(pb.UserRole_value[string(user.Role)]),
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			IsActive:  user.IsActive,
		},
		Profile: &pb.RegisterResponse_TeacherProfile{
			TeacherProfile: &pb.TeacherProfile{
				UserId:     teacher.UserID.String(),
				FullName:   teacher.FullName,
				Department: teacher.Department,
				Position:   teacher.Position,
				TeacherId:  teacher.TeacherID,
			},
		},
	}

	log.Printf("Преподаватель %s успешно зарегистрирован", req.Email)
	return response, nil
}

// Login выполняет вход пользователя в систему
func (s *Server) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	log.Printf("Получен запрос на вход: %s", req.Email)

	// Аутентифицируем пользователя
	user, err := s.userService.AuthenticateUser(ctx, req.Email, req.Password)
	if err != nil {
		log.Printf("Ошибка аутентификации пользователя %s: %v", req.Email, err)
		return nil, status.Errorf(codes.Unauthenticated, "Неверный email или пароль")
	}

	// Генерируем JWT токен
	token, err := s.jwtManager.GenerateToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		log.Printf("Ошибка генерации JWT токена для пользователя %s: %v", user.Email, err)
		return nil, status.Errorf(codes.Internal, "Ошибка генерации токена")
	}

	// Формируем ответ
	response := &pb.LoginResponse{
		Success: true,
		Message: "Вход выполнен успешно",
		Token:   token,
		User: &pb.User{
			Id:        user.ID.String(),
			Email:     user.Email,
			Role:      pb.UserRole(pb.UserRole_value[string(user.Role)]),
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			IsActive:  user.IsActive,
		},
	}

	log.Printf("Пользователь %s успешно вошел в систему", req.Email)
	return response, nil
}

// GetProfile возвращает профиль текущего пользователя
func (s *Server) GetProfile(ctx context.Context, req *pb.GetProfileRequest) (*pb.GetProfileResponse, error) {
	log.Printf("Получен запрос на получение профиля")

	// Проверяем токен
	claims, err := s.jwtManager.ParseToken(req.Token)
	if err != nil {
		log.Printf("Ошибка проверки токена: %v", err)
		return nil, status.Errorf(codes.Unauthenticated, "Неверный токен")
	}

	// Получаем информацию о пользователе
	user, err := s.userService.GetUserByID(ctx, claims.UserID)
	if err != nil {
		log.Printf("Ошибка получения пользователя %s: %v", claims.UserID, err)
		return nil, status.Errorf(codes.NotFound, "Пользователь не найден")
	}

	// Формируем ответ
	response := &pb.GetProfileResponse{
		Success: true,
		Message: "Профиль получен успешно",
		User: &pb.User{
			Id:        user.ID.String(),
			Email:     user.Email,
			Role:      pb.UserRole(pb.UserRole_value[string(user.Role)]),
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			IsActive:  user.IsActive,
		},
	}

	// В зависимости от роли добавляем профиль
	switch user.Role {
	case users.RoleStudent:
		// TODO: Получить профиль студента из БД
		response.Profile = &pb.GetProfileResponse_StudentProfile{
			StudentProfile: &pb.StudentProfile{
				UserId: user.ID.String(),
				// Пока заполняем заглушкой, позже получим реальные данные
			},
		}
	case users.RoleTeacher:
		// TODO: Получить профиль преподавателя из БД
		response.Profile = &pb.GetProfileResponse_TeacherProfile{
			TeacherProfile: &pb.TeacherProfile{
				UserId: user.ID.String(),
				// Пока заполняем заглушкой, позже получим реальные данные
			},
		}
	}

	log.Printf("Профиль пользователя %s успешно получен", user.Email)
	return response, nil
}

// Start запускает gRPC сервер
// Исправленная сигнатура метода
func (s *Server) Start(port int, scheduleService *schedule.Service, userService *users.Service) error {
	// Создаем TCP слушатель
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("ошибка создания TCP слушателя: %w", err)
	}

	// Создаем gRPC сервер
	grpcServer := grpc.NewServer()

	// Регистрируем наши сервисы
	pb.RegisterUserServiceServer(grpcServer, s)

	// Регистрируем Schedule Service
	// Предполагая, что у вас есть функция RegisterService в пакете schedulegrpc
	schedulegrpc.RegisterService(grpcServer, scheduleService, s.jwtManager, userService)

	// Включаем Reflection API для grpcurl и других инструментов
	reflection.Register(grpcServer)

	log.Printf("Запуск gRPC сервера на порту %d", port)

	// Запускаем сервер
	if err := grpcServer.Serve(lis); err != nil {
		return fmt.Errorf("ошибка запуска gRPC сервера: %w", err)
	}

	return nil
}
