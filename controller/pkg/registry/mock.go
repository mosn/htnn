// Copyright The HTNN Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

// fakeServiceEntryStore is a dummy implementation of ServiceEntryStore just for test
type fakeServiceEntryStore struct {
}

func (f *fakeServiceEntryStore) Delete(service string) {
}

func (f *fakeServiceEntryStore) Update(service string, se *ServiceEntryWrapper) {
}

func FakeServiceEntryStore() *fakeServiceEntryStore {
	return &fakeServiceEntryStore{}
}
