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

//go:build envoy1.35 || envoy1.36 || envoy1.37

package api

// SslConnection is not available on Envoy < 1.38.
// Defined here so the StreamInfo interface compiles on these versions.
type SslConnection interface {
	PeerCertificatePresented() bool
	PeerCertificateValidated() bool
	Sha256PeerCertificateDigest() string
	SerialNumberPeerCertificate() string
	SubjectPeerCertificate() string
	IssuerPeerCertificate() string
	SubjectLocalCertificate() string
	UriSanPeerCertificate() []string
	UriSanLocalCertificate() []string
	DnsSansPeerCertificate() []string
	DnsSansLocalCertificate() []string
	ValidFromPeerCertificate() (uint64, bool)
	ExpirationPeerCertificate() (uint64, bool)
	TlsVersion() string
	CiphersuiteString() string
	CiphersuiteId() (uint16, bool)
	SessionId() (string, bool)
	UrlEncodedPemEncodedPeerCertificate() string
	UrlEncodedPemEncodedPeerCertificateChain() string
}
