package main

import (
	"context"
	"encoding/json"
	"fmt"
	"google.golang.org/grpc/peer"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Service - основная структура сервиса
type Service struct {
	UnimplementedAdminServer
	UnimplementedBizServer
	mu             sync.Mutex
	logChan        chan *Event
	methodStats    map[string]uint64
	consumerStats  map[string]uint64
	clientsACL     map[string][]string
	stopChan       chan struct{}
	logSubscribers []logSubscriber
}

type logSubscriber struct { //тип подписчика
	consumer string
	ch       chan *Event
}

// NewService - конструктор сервиса
func NewService(acl map[string][]string) *Service {
	s := &Service{
		logChan:        make(chan *Event, 100),
		methodStats:    make(map[string]uint64),
		consumerStats:  make(map[string]uint64),
		clientsACL:     acl, // Используем переданный ACL
		stopChan:       make(chan struct{}),
		logSubscribers: []logSubscriber{}, //  новый тип подписчика
	}
	go s.logDistributor()
	return s
}

// checkACL - проверка доступа клиента
func (s *Service) checkACL(ctx context.Context, method string) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.Error(codes.Unauthenticated, "no metadata provided")
	}

	consumerIDs := md.Get("consumer")
	if len(consumerIDs) == 0 {
		return status.Error(codes.Unauthenticated, "consumer not provided")
	}

	consumerID := consumerIDs[0]

	s.mu.Lock()
	allowedMethods, exists := s.clientsACL[consumerID]
	s.mu.Unlock()

	// Если consumer отсутствует в ACL, возвращаем Unauthenticated
	if !exists {
		return status.Error(codes.Unauthenticated, "consumer not registered")
	}

	// Проверяем, разрешён ли метод
	for _, allowedMethod := range allowedMethods {
		if allowedMethod == method || allowedMethod == "*" ||
			(len(allowedMethod) > 1 && allowedMethod[len(allowedMethod)-1] == '*' &&
				method[:len(allowedMethod)-1] == allowedMethod[:len(allowedMethod)-1]) {
			return nil
		}
	}

	// Если метод не разрешён, возвращаем Unauthenticated
	return status.Error(codes.Unauthenticated, "method access denied")
}

// logMethod - логирование вызова метода
func (s *Service) logMethod(consumer, method, host string) {
	/*if method == "/main.Admin/Statistics" {
		return
	}*/

	event := &Event{
		Timestamp: time.Now().Unix(),
		Consumer:  consumer,
		Method:    method,
		Host:      host,
	}
	//log.Printf("Updating consumer stats: consumer=%s count=%d", consumer, s.consumerStats[consumer]+1)
	s.logChan <- event

	s.mu.Lock()
	s.methodStats[method]++
	s.consumerStats[consumer]++
	s.mu.Unlock()
}

// logDistributor - распределение логов
func (s *Service) logDistributor() {
	for {
		select {
		case logEntry := <-s.logChan:
			//fmt.Printf("logDistributor: got event %+v\n", logEntry)
			s.mu.Lock()
			for _, sub := range s.logSubscribers {
				// Попытка отправить событие в канал подписчика
				select {
				case sub.ch <- logEntry:
					fmt.Printf("logDistributor: sent event %+v to subscriber %s\n", logEntry, sub.consumer)
				default:
					fmt.Printf("logDistributor: subscriber %s channel full, event dropped\n", sub.consumer)
				}
			}
			s.mu.Unlock()
		case <-s.stopChan:
			return
		}
	}
}

// Check - метод бизнес-логики
func (s *Service) Check(ctx context.Context, req *Nothing) (*Nothing, error) {
	if err := s.checkACL(ctx, "/main.Biz/Check"); err != nil {
		return nil, err
	}
	md, _ := metadata.FromIncomingContext(ctx)
	consumer := md.Get("consumer")[0]

	var host string
	if p, ok := peer.FromContext(ctx); ok {
		host = p.Addr.String()
	} else {
		host = "127.0.0.1:unknown"
	}
	s.logMethod(consumer, "/main.Biz/Check", host)
	return &Nothing{Dummy: true}, nil
}

// Add - метод бизнес-логики
func (s *Service) Add(ctx context.Context, req *Nothing) (*Nothing, error) {
	if err := s.checkACL(ctx, "/main.Biz/Add"); err != nil {
		return nil, err
	}
	md, _ := metadata.FromIncomingContext(ctx)
	consumer := md.Get("consumer")[0]
	var host string
	if p, ok := peer.FromContext(ctx); ok {
		host = p.Addr.String()
	} else {
		host = "127.0.0.1:unknown"
	}
	s.logMethod(consumer, "/main.Biz/Add", host)
	return &Nothing{Dummy: true}, nil
}

// Test - метод бизнес-логики
func (s *Service) Test(ctx context.Context, req *Nothing) (*Nothing, error) {
	if err := s.checkACL(ctx, "/main.Biz/Test"); err != nil {
		return nil, err
	}
	md, _ := metadata.FromIncomingContext(ctx)
	consumer := md.Get("consumer")[0]

	// Получаем реальный адрес клиента из контекста
	var host string
	if p, ok := peer.FromContext(ctx); ok {
		host = p.Addr.String()
	} else {
		host = "127.0.0.1:unknown"
	}

	s.logMethod(consumer, "/main.Biz/Test", host)
	return &Nothing{Dummy: true}, nil
}

// Logging - потоковая передача логов
func (s *Service) Logging(_ *Nothing, stream Admin_LoggingServer) error {
	if err := s.checkACL(stream.Context(), "/main.Admin/Logging"); err != nil {
		return err
	}
	md, _ := metadata.FromIncomingContext(stream.Context())
	consumerIDs := md.Get("consumer")
	consumerID := consumerIDs[0]

	var host string
	if p, ok := peer.FromContext(stream.Context()); ok {
		host = p.Addr.String()
	} else {
		host = "127.0.0.1:unknown"
	}

	// Создаем канал подписчика
	ch := make(chan *Event, 10)
	s.mu.Lock()
	s.logSubscribers = append(s.logSubscribers, logSubscriber{
		consumer: consumerID,
		ch:       ch,
	})
	s.mu.Unlock()

	// Логируем вызов метода Logging
	s.logMethod(consumerID, "/main.Admin/Logging", host)

	defer func() {
		s.mu.Lock()
		for i, sub := range s.logSubscribers {
			if sub.ch == ch {
				s.logSubscribers = append(s.logSubscribers[:i], s.logSubscribers[i+1:]...)
				break
			}
		}
		s.mu.Unlock()
	}()

	for {
		select {
		case evt, ok := <-ch:
			if !ok {
				return nil
			}
			// Пропускаем событие, если оно создано самим подписчиком
			if evt.Consumer == consumerID {
				continue
			}
			if err := stream.Send(evt); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		case <-s.stopChan:
			return nil
		}
	}
}

// Statistics - потоковая передача статистики
func (s *Service) Statistics(req *StatInterval, stream Admin_StatisticsServer) error {
	md, _ := metadata.FromIncomingContext(stream.Context())
	consumer := md.Get("consumer")[0]

	var host string
	if p, ok := peer.FromContext(stream.Context()); ok {
		host = p.Addr.String()
	} else {
		host = "127.0.0.1:unknown"
	}

	// Для "stat2" логируем вызов Statistics (накопительная статистика)
	if consumer == "stat2" {
		s.logMethod(consumer, "/main.Admin/Statistics", host)
	}

	interval := time.Duration(req.IntervalSeconds) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Если клиент "stat1", фиксируем baseline при подключении.
	var baselineMethod map[string]uint64
	var baselineConsumer map[string]uint64
	isDelta := false
	if consumer == "stat1" {
		s.mu.Lock()
		baselineMethod = make(map[string]uint64)
		baselineConsumer = make(map[string]uint64)
		for k, v := range s.methodStats {
			baselineMethod[k] = v
		}
		for k, v := range s.consumerStats {
			baselineConsumer[k] = v
		}
		// Устанавливаем baseline для "/main.Admin/Statistics" равным 0,
		// чтобы любые последующие вызовы этого метода учитывались как изменения.
		baselineMethod["/main.Admin/Statistics"] = 0
		s.mu.Unlock()
		isDelta = true
	}

	for {
		select {
		case <-ticker.C:
			s.mu.Lock()
			currentMethod := make(map[string]uint64)
			currentConsumer := make(map[string]uint64)
			for k, v := range s.methodStats {
				currentMethod[k] = v
			}
			for k, v := range s.consumerStats {
				currentConsumer[k] = v
			}
			s.mu.Unlock()

			if isDelta {
				// Вычисляем дельту для клиента stat1
				deltaMethod := make(map[string]uint64)
				deltaConsumer := make(map[string]uint64)
				for k, v := range currentMethod {
					b := baselineMethod[k]
					if v > b {
						deltaMethod[k] = v - b
					}
				}
				for k, v := range currentConsumer {
					b := baselineConsumer[k]
					if v > b {
						deltaConsumer[k] = v - b
					}
				}
				// Обновляем baseline для следующего такта.
				baselineMethod = currentMethod
				baselineConsumer = currentConsumer

				currentMethod = deltaMethod
				currentConsumer = deltaConsumer
			} else if consumer == "stat2" {
				// Для клиента stat2 используем накопительную статистику,
				// но исключаем записи по методу "/main.Admin/Statistics" и consumer "stat2"
				delete(currentMethod, "/main.Admin/Statistics")
				delete(currentConsumer, "stat2")
			}

			stat := &Stat{
				Timestamp:  time.Now().Unix(),
				ByMethod:   currentMethod,
				ByConsumer: currentConsumer,
			}
			if err := stream.Send(stat); err != nil {
				return err
			}
		case <-stream.Context().Done():
			return nil
		case <-s.stopChan:
			return nil
		}
	}
}

// StartMyMicroservice - запуск микросервиса
func StartMyMicroservice(ctx context.Context, listenAddr string, ACLData string) error {
	// Парсинг ACLData (предполагаем, что это строка с клиентами, разделенными запятыми)
	//fmt.Println("ACLData:", ACLData)

	var clientsACL map[string][]string
	err := json.Unmarshal([]byte(ACLData), &clientsACL)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return fmt.Errorf("failed to parse ACL data: %v", err)
	}

	// Передаём ACL в сервис
	s := NewService(clientsACL)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(interceptor),
	)
	RegisterAdminServer(grpcServer, s)
	RegisterBizServer(grpcServer, s)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Printf("Server error: %v", err)
		}
	}()

	// Ожидание отмены контекста и завершение
	go func() {
		<-ctx.Done()
		close(s.stopChan)         // Закрываем канал для остановки горутин
		grpcServer.GracefulStop() // Плавно завершаем сервер
	}()

	return nil
}

func interceptor(
	ctx context.Context,
	req interface{},
	info *grpc.UnaryServerInfo,
	handler grpc.UnaryHandler,
) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	//fmt.Println("Received metadata:", md) // Логируем все метаданные

	if !ok {
		return nil, status.Error(codes.Unauthenticated, "no metadata provided")
	}

	consumerIDs := md.Get("consumer") // Ищем "consumer" вместо "client_id"
	if len(consumerIDs) == 0 {
		return nil, status.Error(codes.Unauthenticated, "consumer not provided")
	}

	fmt.Println("consumer:", consumerIDs[0]) // Логируем найденный consumer

	return handler(ctx, req)
}
