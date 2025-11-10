package core

import (
	"context"
	"fmt"
	"log/slog"
	"slices"
	"sync"
)

type Service struct {
	log         *slog.Logger
	db          DB
	xkcd        XKCD
	words       Words
	concurrency int
	status      ServiceStatus
}

func NewService(
	log *slog.Logger, db DB, xkcd XKCD, words Words, concurrency int,
) (*Service, error) {
	if concurrency < 1 {
		return nil, fmt.Errorf("wrong concurrency specified: %d", concurrency)
	}
	return &Service{
		log:         log,
		db:          db,
		xkcd:        xkcd,
		words:       words,
		concurrency: concurrency,
		status:      StatusIdle,
	}, nil
}

func (s *Service) Update(ctx context.Context) (err error) {
	s.status = StatusRunning
	defer func() { s.status = StatusIdle }()
	var ids []int
	existIDs, err := s.db.IDs(ctx)
	if err != nil {
		return err
	}
	lastID, err := s.xkcd.LastID(ctx)
	if err != nil {
		return err
	}
	for i := range lastID + 1 {
		if !slices.Contains(existIDs, i) && i != 404 && i != 0 {
			ids = append(ids, i)
		}
	}
	for i := 0; i < len(ids); i += s.concurrency {
		end := i + s.concurrency
		if end > len(ids) {
			end = len(ids)
		}
		var wg sync.WaitGroup
		comicsChan := make(chan Comics, s.concurrency)
		for _, id := range ids[i:end] {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				xkcdComics, err := s.xkcd.Get(ctx, id)
				if err != nil {
					s.log.Error("xkcd.get error", "error", err)
					return
				}
				words, err := s.words.Norm(ctx, xkcdComics.Title+" "+xkcdComics.Description)
				if err != nil {
					s.log.Error("words.norm error", "error", err)
					return
				}
				wordsMap := make(map[string]int)
				for _, w := range words {
					wordsMap[w]++
				}
				comics := Comics{
					ID:    xkcdComics.ID,
					URL:   xkcdComics.URL,
					Words: wordsMap,
				}

				comicsChan <- comics
			}(id)
		}
		go func() {
			wg.Wait()
			close(comicsChan)
		}()
		for c := range comicsChan {
			if err := s.db.Add(ctx, c); err != nil {
				s.log.Error("db.add error", "error", err)
			}
		}
	}
	return nil
}

func (s *Service) Stats(ctx context.Context) (ServiceStats, error) {
	dbStats, err := s.db.Stats(ctx)
	if err != nil {
		s.log.Error("db.stats", "error", err)
		return ServiceStats{}, err
	}
	comicsTotal, err := s.xkcd.LastID(ctx)
	if err != nil {
		s.log.Error("xkcd.lastid", "error", err)
		return ServiceStats{}, err
	}
	return ServiceStats{
		DBStats:     dbStats,
		ComicsTotal: comicsTotal - 1,
	}, nil
}

func (s *Service) Status(ctx context.Context) ServiceStatus {
	return s.status
}

func (s *Service) Drop(ctx context.Context) error {
	if err := s.db.Drop(ctx); err != nil {
		s.log.Error("db.drop", "error", err)
		return err
	}
	return nil
}
