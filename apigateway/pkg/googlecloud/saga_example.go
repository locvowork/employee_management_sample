package googlecloud

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// =============================================================================
// SAGA PATTERN EXAMPLE: ORDER PROCESSING
// =============================================================================
//
// This example demonstrates a typical e-commerce order processing saga:
// 1. Reserve Inventory
// 2. Process Payment
// 3. Create Shipment
// 4. Send Notification
//
// If any step fails, previous steps are rolled back (compensated).
// =============================================================================

// --- Example Domain Models ---

// OrderPayload represents the initial order data
type OrderPayload struct {
	OrderID    string  `json:"order_id"`
	CustomerID string  `json:"customer_id"`
	ProductID  string  `json:"product_id"`
	Quantity   int     `json:"quantity"`
	Amount     float64 `json:"amount"`
}

// ReservationResult represents the output of inventory reservation
type ReservationResult struct {
	ReservationID string `json:"reservation_id"`
}

// PaymentResult represents the output of payment processing
type PaymentResult struct {
	TransactionID string `json:"transaction_id"`
}

// ShipmentResult represents the output of shipment creation
type ShipmentResult struct {
	TrackingNumber string `json:"tracking_number"`
}

// --- Example Step Implementations ---

// ReserveInventoryStep reserves inventory for an order
func ReserveInventoryStep(ctx context.Context, input string) (string, error) {
	var order OrderPayload
	if err := json.Unmarshal([]byte(input), &order); err != nil {
		return "", err
	}

	// Simulate inventory reservation
	// In a real system, this would call an inventory service
	fmt.Printf("[Inventory] Reserving %d units of product %s\n", order.Quantity, order.ProductID)

	// Simulate some processing time
	time.Sleep(100 * time.Millisecond)

	result := ReservationResult{
		ReservationID: fmt.Sprintf("RES-%s-%d", order.OrderID, time.Now().Unix()),
	}

	output, err := json.Marshal(result)
	return string(output), err
}

// CancelInventoryReservation compensates for ReserveInventoryStep
func CancelInventoryReservation(ctx context.Context, input, output string) error {
	var result ReservationResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return err
	}

	fmt.Printf("[Inventory] Cancelling reservation %s\n", result.ReservationID)
	// In a real system, this would call the inventory service to release the reservation
	return nil
}

// ProcessPaymentStep processes payment for an order
func ProcessPaymentStep(ctx context.Context, input string) (string, error) {
	// The input here could be the reservation result or original order
	var reservation ReservationResult
	if err := json.Unmarshal([]byte(input), &reservation); err != nil {
		// Try parsing as original order
		var order OrderPayload
		if err := json.Unmarshal([]byte(input), &order); err != nil {
			return "", err
		}
		fmt.Printf("[Payment] Processing payment of $%.2f for order %s\n", order.Amount, order.OrderID)
	}

	// Simulate payment processing
	time.Sleep(150 * time.Millisecond)

	// Uncomment to simulate payment failure:
	// return "", fmt.Errorf("payment declined: insufficient funds")

	result := PaymentResult{
		TransactionID: fmt.Sprintf("TXN-%d", time.Now().Unix()),
	}

	output, err := json.Marshal(result)
	return string(output), err
}

// RefundPayment compensates for ProcessPaymentStep
func RefundPayment(ctx context.Context, input, output string) error {
	var result PaymentResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return err
	}

	fmt.Printf("[Payment] Refunding transaction %s\n", result.TransactionID)
	// In a real system, this would call the payment service to issue a refund
	return nil
}

// CreateShipmentStep creates a shipment for delivery
func CreateShipmentStep(ctx context.Context, input string) (string, error) {
	fmt.Printf("[Shipment] Creating shipment for order\n")

	// Simulate shipment creation
	time.Sleep(100 * time.Millisecond)

	result := ShipmentResult{
		TrackingNumber: fmt.Sprintf("TRACK-%d", time.Now().Unix()),
	}

	output, err := json.Marshal(result)
	return string(output), err
}

// CancelShipment compensates for CreateShipmentStep
func CancelShipment(ctx context.Context, input, output string) error {
	var result ShipmentResult
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return err
	}

	fmt.Printf("[Shipment] Cancelling shipment %s\n", result.TrackingNumber)
	return nil
}

// SendNotificationStep sends order confirmation notification
func SendNotificationStep(ctx context.Context, input string) (string, error) {
	fmt.Printf("[Notification] Sending order confirmation\n")

	// Notifications are typically not compensated
	time.Sleep(50 * time.Millisecond)

	return `{"notified": true}`, nil
}

// --- Example Orchestrator Factory ---

// NewOrderProcessingSaga creates a saga orchestrator for order processing
func NewOrderProcessingSaga(client *Client) *SagaOrchestrator {
	steps := []SagaStepDefinition{
		{
			Name:        "Reserve Inventory",
			ServiceName: "inventory-service",
			Execute:     ReserveInventoryStep,
			Compensate:  CancelInventoryReservation,
		},
		{
			Name:        "Process Payment",
			ServiceName: "payment-service",
			Execute:     ProcessPaymentStep,
			Compensate:  RefundPayment,
		},
		{
			Name:        "Create Shipment",
			ServiceName: "shipping-service",
			Execute:     CreateShipmentStep,
			Compensate:  CancelShipment,
		},
		{
			Name:        "Send Notification",
			ServiceName: "notification-service",
			Execute:     SendNotificationStep,
			Compensate:  nil, // No compensation for notifications
		},
	}

	return NewSagaOrchestrator(client, steps)
}

// --- Example Usage ---

// ExampleProcessOrder demonstrates how to use the saga pattern
func ExampleProcessOrder(ctx context.Context, client *Client, order OrderPayload) error {
	// Create the orchestrator
	orchestrator := NewOrderProcessingSaga(client)

	// Convert order to JSON payload
	payload, err := json.Marshal(order)
	if err != nil {
		return err
	}

	// Generate a unique saga ID
	sagaID := fmt.Sprintf("ORDER-SAGA-%s-%d", order.OrderID, time.Now().Unix())

	// Start the saga
	if err := orchestrator.Start(ctx, sagaID, "Order Processing", string(payload)); err != nil {
		return fmt.Errorf("failed to start saga: %w", err)
	}

	// Execute the saga
	if err := orchestrator.Execute(ctx, sagaID); err != nil {
		return fmt.Errorf("saga execution failed: %w", err)
	}

	return nil
}

// --- Recovery Worker Example ---

// SagaRecoveryWorker periodically checks for interrupted sagas and resumes them
type SagaRecoveryWorker struct {
	client        *Client
	orchestrators map[string]*SagaOrchestrator // Map saga name to orchestrator
	interval      time.Duration
}

// NewSagaRecoveryWorker creates a new recovery worker
func NewSagaRecoveryWorker(client *Client, interval time.Duration) *SagaRecoveryWorker {
	return &SagaRecoveryWorker{
		client:        client,
		orchestrators: make(map[string]*SagaOrchestrator),
		interval:      interval,
	}
}

// RegisterOrchestrator registers an orchestrator for a saga type
func (w *SagaRecoveryWorker) RegisterOrchestrator(name string, o *SagaOrchestrator) {
	w.orchestrators[name] = o
}

// Run starts the recovery worker
func (w *SagaRecoveryWorker) Run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.recoverSagas(ctx)
		}
	}
}

func (w *SagaRecoveryWorker) recoverSagas(ctx context.Context) {
	sagas, err := w.client.ListPendingSagas(ctx, 10)
	if err != nil {
		fmt.Printf("Failed to list pending sagas: %v\n", err)
		return
	}

	for _, saga := range sagas {
		orchestrator, ok := w.orchestrators[saga.Name]
		if !ok {
			fmt.Printf("No orchestrator registered for saga type: %s\n", saga.Name)
			continue
		}

		fmt.Printf("Resuming saga %s (type: %s)\n", saga.ID, saga.Name)
		if err := orchestrator.Resume(ctx, saga.ID); err != nil {
			fmt.Printf("Failed to resume saga %s: %v\n", saga.ID, err)
		}
	}
}
