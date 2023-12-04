/*
 *    Copyright 2023 wkRonin
 *
 *    Licensed under the Apache License, Version 2.0 (the "License");
 *    you may not use this file except in compliance with the License.
 *    You may obtain a copy of the License at
 *
 *        http://www.apache.org/licenses/LICENSE-2.0
 *
 *    Unless required by applicable law or agreed to in writing, software
 *    distributed under the License is distributed on an "AS IS" BASIS,
 *    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *    See the License for the specific language governing permissions and
 *    limitations under the License.
 */

package saramax

import (
	"context"
	"encoding/json"
	"time"

	"github.com/IBM/sarama"

	"github.com/wkRonin/toolkit/logger"
)

type BatchHandler[T any] struct {
	l         logger.Logger
	fn        func(msgs []*sarama.ConsumerMessage, t []T) error
	batchSize int
}

// NewBatchHandler 实现了ConsumerGroupHandler接口的handler:批量消费后再提交
func NewBatchHandler[T any](l logger.Logger,
	fn func(msgs []*sarama.ConsumerMessage, t []T) error,
	batchSize int) *BatchHandler[T] {
	return &BatchHandler[T]{
		l:         l,
		fn:        fn,
		batchSize: batchSize,
	}
}

func (h *BatchHandler[T]) Setup(session sarama.ConsumerGroupSession) error {
	return nil
}

func (h *BatchHandler[T]) Cleanup(session sarama.ConsumerGroupSession) error {
	return nil
}

// ConsumeClaim 批量消费
func (h *BatchHandler[T]) ConsumeClaim(session sarama.ConsumerGroupSession,
	claim sarama.ConsumerGroupClaim) error {
	msgsCh := claim.Messages()
	for {
		msgs := make([]*sarama.ConsumerMessage, 0, h.batchSize)
		ts := make([]T, 0, h.batchSize)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		done := false
		for i := 0; i < h.batchSize && !done; i++ {
			select {
			case <-ctx.Done():
				done = true
			case msg, ok := <-msgsCh:
				if !ok {
					cancel()
					// channel 被关闭了
					return nil
				}
				var t T
				err := json.Unmarshal(msg.Value, &t)
				if err != nil {
					h.l.Error("反序列化消息体失败",
						logger.String("topic", msg.Topic),
						logger.Int32("partition", msg.Partition),
						logger.Int64("offset", msg.Offset),
						// 这里可以考虑打印 msg.Value，但是有些时候 msg 本身包含敏感数据
						logger.Error(err))
					continue
				}
				msgs = append(msgs, msg)
				ts = append(ts, t)
			}
		}
		cancel()
		if len(msgs) == 0 {
			continue
		}
		err := h.fn(msgs, ts)
		if err != nil {
			h.l.Error("批量消费失败", logger.Any("ConsumerMessages", msgs))
		}
		if err == nil {
			// 遍历提交
			for _, msg := range msgs {
				session.MarkMessage(msg, "")
			}
		}
	}
}
