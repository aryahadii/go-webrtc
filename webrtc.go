/*
Package webrtc is a golang wrapper for libwebrtc.

To provide an easier experience for users of this package, there are differences
inherent in the interface written here and the original native code WebRTC. This
allows users to use WebRTC in a more idiomatic golang way. For example, callback
mechanism has a layer of indirection that allows goroutines instead.

The interface here is based mostly on: w3c.github.io/webrtc-pc

There is also a complication in building the dependent static library for this
to work. Furthermore it is possible that this will break on future versions
of libwebrtc, because the interface with the native code is be fragile.

Latest tested / compatible version of webrtc HEAD: cb3f9bd
More later...

TODO(keroserene): More package documentation, and more documentation in general.
*/
package webrtc

/*
#cgo CPPFLAGS: -Ithird_party/libwebrtc/
#cgo CXXFLAGS: -std=c++0x
// -Wno-c++0x-extensions
#cgo LDFLAGS: -Wl,-z,now -Wl,-z,relro -Wl,--fatal-warnings -Wl,-z,defs -pthread
#cgo LDFLAGS: -Wl,-z,noexecstack -fPIC -fuse-ld=gold
#cgo LDFLAGS: -Bbin
#cgo LDFLAGS: -Wl,--disable-new-dtags -pthread -m64 -Wl,--detect-odr-violations
// #cgo LDFLAGS: -Wl,--icf=all -Wl,-O1 -Wl,--as-needed -Wl,--gc-sections
#cgo LDFLAGS: -Llib
// libwebrtc_magic is a custom built archive based on many other ninja files.
#cgo LDFLAGS: -lwebrtc_magic
#cgo LDFLAGS: -lgthread-2.0 -lgtk-x11-2.0 -lgdk-x11-2.0 -lpangocairo-1.0
#cgo LDFLAGS: -latk-1.0 -lcairo -lgdk_pixbuf-2.0 -lgio-2.0 -lpangoft2-1.0 -lpango-1.0
#cgo LDFLAGS: -lgobject-2.0 -lglib-2.0 -lfontconfig -lfreetype -lX11 -lXcomposite
#cgo LDFLAGS: -lXext -lXrender -ldl -lrt -lexpat -lm
#include "cpeerconnection.h"
*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

type SDPHeader struct {
	// Keep track of both a pointer to the C++ SessionDescription object,
	// and the serialized string version (which native code generates)
	cgoSdp      C.CGOsdp
	description string
}

type RTCConfiguration struct {
	// TODO: Implement, and provide as argument to CreatePeerConnection
	IceServers           []string
	IceTransportPolicy   string
	BundlePolicy         string
	RtcpMuxPolicy        string
	PeerIdentity         string // Target peer identity
	Certificates         []string // TODO: implement to allow key continuity
	IceCandidatePoolSize int

	// Internal RTCConfiguration for C
	cgoConfig  *C.CGORTCConfiguration
}

// Create a new RTCConfiguration with spec default values.
func NewRTCConfiguration() *RTCConfiguration {
	c := new(RTCConfiguration)
	// c.IceServers = make([]string, 0)
	c.IceServers = nil
	c.IceTransportPolicy = "all"
	c.BundlePolicy = "balanced"
	c.RtcpMuxPolicy = "require"
	c.Certificates = make([]string, 0)
	return c
}
func (config *RTCConfiguration) CGO() C.CGORTCConfiguration{
	c := new(C.CGORTCConfiguration)
	// c.IceServers = unsafe.Pointer(&config.IceServers)
	c.IceTransportPolicy = C.CString(config.IceTransportPolicy)
	c.BundlePolicy = C.CString(config.BundlePolicy)
	c.RtcpMuxPolicy = C.CString(config.RtcpMuxPolicy)
	c.PeerIdentity = C.CString(config.PeerIdentity)
	// c.Certificates = config.Certificates
	c.IceCandidatePoolSize = C.int(config.IceCandidatePoolSize)
	config.cgoConfig = c
	return *c
}

type PeerConnection struct {

	// currentLocalDescription
	// pendingLocalDescription
	// setRemoteDescription
	localDescription  *SDPHeader
	remoteDescription *SDPHeader
	// remoteDescription
	// currentRemoteDescription
	// pendingRemoteDescription
	// addIceCandidate
	// signalingState
	// iceGatheringState
	// iceConnectionState
	// canTrickleIceCandidates
	// getConfiguration
	// setConfiguration
	// close
	// onnegotiationneeded
	OnIceCandidate func()
	// onsignalingstatechange
	// oniceconnectionstatechange
	// onicegatheringstatechange
	IceServers string

	// Internal PeerConnection functionality.
	cgoPeer C.CGOPeer
}

type DataChannel struct {
}

type DataChannelInit struct {
}

// PeerConnection constructor.
func NewPeerConnection(config *RTCConfiguration) (*PeerConnection, error) {
	pc := new(PeerConnection)
	pc.cgoPeer = C.CGOInitializePeer()  // internal CGO Peer.
	if nil == pc.cgoPeer {
		return pc, errors.New("PeerConnection: failed to initialize.")
	}
	// Convert config to C struct.
	// cConfig := (*C.CGORTCConfiguration)(unsafe.Pointer(&config))
	// cConfig. C.CString(config.IceServers)
	cConfig := config.CGO()
	if 0 != C.CGOCreatePeerConnection(pc.cgoPeer, &cConfig) {
		return nil, errors.New("PeerConnection: could not create from config.")
	}
	return pc, nil
}

// CreateOffer prepares an SDP "offer" message, which should be sent to the target
// peer over a signalling channel.
func (pc *PeerConnection) CreateOffer() (*SDPHeader, error) {
	sdp := C.CGOCreateOffer(pc.cgoPeer)
	if nil == sdp {
		return nil, errors.New("CreateOffer: could not prepare SDP offer.")
	}
	offer := new(SDPHeader)
	offer.cgoSdp = sdp
	offer.description = C.GoString(C.CGOSerializeSDP(sdp))
	return offer, nil
}

func (pc *PeerConnection) SetLocalDescription(sdp *SDPHeader) error {
	r := C.CGOSetLocalDescription(pc.cgoPeer, sdp.cgoSdp)
	if 0 != r {
		return errors.New("SetLocalDescription failed.")
	}
	pc.localDescription = sdp
	return nil
}

// readonly localDescription
func (pc *PeerConnection) LocalDescription() (sdp *SDPHeader) {
	return pc.localDescription
}

func (pc *PeerConnection) SetRemoteDescription(sdp *SDPHeader) error {
	r := C.CGOSetRemoteDescription(pc.cgoPeer, sdp.cgoSdp)
	if 0 != r {
		return errors.New("SetRemoteDescription failed.")
	}
	pc.remoteDescription = sdp
	return nil
}

// readonly remoteDescription
func (pc *PeerConnection) RemoteDescription() (sdp *SDPHeader) {
	return pc.remoteDescription
}

// CreateAnswer prepares an SDP "answer" message, which should be sent in
// response to a peer that has sent an offer, over the signalling channel.
func (pc *PeerConnection) CreateAnswer() (*SDPHeader, error) {
	fmt.Println("[go] creating answer...")
	sdp := C.CGOCreateAnswer(pc.cgoPeer)
	if nil == sdp {
		return nil, errors.New("[C ERROR] CreateAnswer - could not prepare SDP offer.")
	}
	answer := new(SDPHeader)
	answer.cgoSdp = sdp
	answer.description = C.GoString(C.CGOSerializeSDP(sdp))
	return answer, nil
}

// TODO: Above methods blocks until success or failure occurs. Maybe there should
// actually be a callback version, so the user doesn't have to make their own
// goroutine.

func (pc *PeerConnection) CreateDataChannel(label string, dict DataChannelInit) (*DataChannel, error) {
	C.CGOCreateDataChannel(pc.cgoPeer, C.CString(label), unsafe.Pointer(&dict))
	d := new(DataChannel)
	return d, nil
}

// Install a handler for receiving ICE Candidates.
// func OnIceCandidate(pc PeerConnection) {
// }

type RTCSignalingState int

/*
const {
  stable RTCSignallingState = iota
  have-local-offer
  have-remote-offer
  have-local-pranswer
  have-remote-pranswer
  closed
}
*/

type RTCIceCredentialType struct {
}

type RTCIceServer struct {
	Urls       string
	Username   string
	Credential string
	// credentialType   RTCIceCredentialType
}

type RTCIceTransportPolicy struct {
}

type RTCBundlePolicy struct {
}
