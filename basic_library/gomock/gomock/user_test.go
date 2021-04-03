package user

import (
	"testing"

	mock_user "sample/mock_user"

	"github.com/golang/mock/gomock"
)

func TestRemember(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIndex := mock_user.NewMockIndex(ctrl)
	mockIndex.EXPECT().Put("a", 1)
	mockIndex.EXPECT().Put("b", gomock.Eq(2))
	mockIndex.EXPECT().NillableRet()

	boolc := make(chan bool)
	mockIndex.EXPECT().ConcreteRet().Return(boolc)
	mockIndex.EXPECT().ConcreteRet().Return(nil)
	mockIndex.EXPECT().Ellip("%d", 0, 1, 1, 2, 3)
	tri := []interface{}{1, 3, 6, 10, 15}
	mockIndex.EXPECT().Ellip("%d", tri...)
	mockIndex.EXPECT().EllipOnly(gomock.Eq("arg"))
	Remember(mockIndex, []string{"a", "b"}, []interface{}{1, 2})
	if c := mockIndex.ConcreteRet(); c != boolc {
		t.Errorf("ConcreteRet: got %v, want %v", c, boolc)
	}
	if c := mockIndex.ConcreteRet(); c != nil {
		t.Errorf("ConcreteRet: got %v, want nil", c)
	}

	calledString := ""
	mockIndex.EXPECT().Put(gomock.Any(), gomock.Any()).Do(func(key string, _ interface{}) {
		calledString = key
	})
	mockIndex.EXPECT().NillableRet()
	Remember(mockIndex, []string{"blah"}, []interface{}{7})
	if calledString != "blah" {
		t.Fatalf(`Uh oh. %q != "blah"`, calledString)
	}

	mockIndex.EXPECT().Put("nil-key", gomock.Any()).Do(func(key string, value interface{}) {
		if value != nil {
			t.Errorf("Put did not pass through nil; got %v", value)
		}
	})
	mockIndex.EXPECT().NillableRet()
	Remember(mockIndex, []string{"nil-key"}, []interface{}{nil})
}

func TestVariadicFunction(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockIndex := mock_user.NewMockIndex(ctrl)
	mockIndex.EXPECT().Ellip("%d", 5, 6, 7, 8).Do(func(format string, nums ...int) {
		sum := 0
		for _, value := range nums {
			sum += value
		}
		if sum != 26 {
			t.Errorf("Expected 7, got %d", sum)
		}
	})

	mockIndex.EXPECT().Ellip("%d", gomock.Any()).Do(func(format string, nums ...int) {
		sum := 0
		for _, value := range nums {
			sum += value
		}
		if sum != 10 {
			t.Errorf("Expected 7, got %d", sum)
		}
	})
	mockIndex.EXPECT().Ellip("%d", gomock.Any()).Do(func(format string, nums ...int) {
		sum := 0
		for _, value := range nums {
			sum += value
		}
		if sum != 0 {
			t.Errorf("Expected 0, got %d", sum)
		}
	})
	mockIndex.EXPECT().Ellip("%d", gomock.Any()).Do(func(format string, nums ...int) {
		sum := 0
		for _, value := range nums {
			sum += value
		}
		if sum != 0 {
			t.Errorf("Expected 0, got %d", sum)
		}
	})
	mockIndex.EXPECT().Ellip("%d").Do(func(format string, nums ...int) {
		sum := 0
		for _, value := range nums {
			sum += value
		}
		if sum != 0 {
			t.Errorf("Expected 0, got %d", sum)
		}
	})

	mockIndex.Ellip("%d", 1, 2, 3, 4)
	mockIndex.Ellip("%d", 5, 6, 7, 8)
	mockIndex.Ellip("%d", 0)
	mockIndex.Ellip("%d")
	mockIndex.Ellip("%d")
}
