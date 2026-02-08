package collab

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"strings"
	"time"
)

// BCF 2.1 XML structures for export/import

type bcfVersion struct {
	XMLName   xml.Name `xml:"Version"`
	VersionID string   `xml:"VersionId,attr"`
	XMLNS     string   `xml:"xmlns,attr"`
}

type bcfProject struct {
	XMLName   xml.Name `xml:"ProjectExtension"`
	XMLNS     string   `xml:"xmlns,attr"`
	Project   bcfProjectInfo
}

type bcfProjectInfo struct {
	XMLName   xml.Name `xml:"Project"`
	ProjectID string   `xml:"ProjectId,attr"`
	Name      string   `xml:",chardata"`
}

type bcfMarkup struct {
	XMLName    xml.Name       `xml:"Markup"`
	XMLNS      string         `xml:"xmlns,attr"`
	Topic      bcfTopicXML    `xml:"Topic"`
	Comment    []bcfCommentXML `xml:"Comment"`
	Viewpoints []bcfViewpointRef `xml:"Viewpoints"`
}

type bcfTopicXML struct {
	XMLName        xml.Name `xml:"Topic"`
	GUID           string   `xml:"Guid,attr"`
	TopicType      string   `xml:"TopicType,attr,omitempty"`
	TopicStatus    string   `xml:"TopicStatus,attr,omitempty"`
	Title          string   `xml:"Title"`
	Description    string   `xml:"Description,omitempty"`
	Priority       string   `xml:"Priority,omitempty"`
	CreationDate   string   `xml:"CreationDate"`
	CreationAuthor string   `xml:"CreationAuthor,omitempty"`
	ModifiedDate   string   `xml:"ModifiedDate,omitempty"`
	DueDate        string   `xml:"DueDate,omitempty"`
	AssignedTo     string   `xml:"AssignedTo,omitempty"`
	Stage          string   `xml:"Stage,omitempty"`
	Labels         []string `xml:"Labels,omitempty"`
}

type bcfCommentXML struct {
	XMLName      xml.Name `xml:"Comment"`
	GUID         string   `xml:"Guid,attr"`
	Date         string   `xml:"Date"`
	Author       string   `xml:"Author,omitempty"`
	Comment      string   `xml:"Comment"`
	ViewpointGUID string  `xml:"Viewpoint>Guid,omitempty"`
}

type bcfViewpointRef struct {
	XMLName  xml.Name `xml:"Viewpoints"`
	GUID     string   `xml:"Guid,attr"`
	Viewpoint string  `xml:"Viewpoint"`
	Snapshot  string  `xml:"Snapshot,omitempty"`
}

type bcfVisInfo struct {
	XMLName         xml.Name             `xml:"VisualizationInfo"`
	XMLNS           string               `xml:"xmlns,attr"`
	GUID            string               `xml:"Guid,attr"`
	PerspectiveCamera *bcfPerspective    `xml:"PerspectiveCamera,omitempty"`
	OrthogonalCamera  *bcfOrthogonal     `xml:"OrthogonalCamera,omitempty"`
	ClippingPlanes    *bcfClippingPlanes `xml:"ClippingPlanes,omitempty"`
	Components        *bcfComponentsXML  `xml:"Components,omitempty"`
}

type bcfPerspective struct {
	CameraViewPoint bcfPoint `xml:"CameraViewPoint"`
	CameraDirection bcfPoint `xml:"CameraDirection"`
	CameraUpVector  bcfPoint `xml:"CameraUpVector"`
	FieldOfView     float64  `xml:"FieldOfView"`
}

type bcfOrthogonal struct {
	CameraViewPoint  bcfPoint `xml:"CameraViewPoint"`
	CameraDirection  bcfPoint `xml:"CameraDirection"`
	CameraUpVector   bcfPoint `xml:"CameraUpVector"`
	ViewToWorldScale float64  `xml:"ViewToWorldScale"`
}

type bcfPoint struct {
	X float64 `xml:"X"`
	Y float64 `xml:"Y"`
	Z float64 `xml:"Z"`
}

type bcfClippingPlanes struct {
	ClippingPlane []bcfClippingPlaneXML `xml:"ClippingPlane"`
}

type bcfClippingPlaneXML struct {
	Location  bcfPoint `xml:"Location"`
	Direction bcfPoint `xml:"Direction"`
}

type bcfComponentsXML struct {
	Selection  *bcfComponentSelection  `xml:"Selection,omitempty"`
	Visibility *bcfComponentVisibility `xml:"Visibility,omitempty"`
}

type bcfComponentSelection struct {
	Component []bcfComponentXML `xml:"Component"`
}

type bcfComponentVisibility struct {
	DefaultVisibility bool              `xml:"DefaultVisibility,attr"`
	Exceptions        []bcfComponentXML `xml:"Exceptions>Component,omitempty"`
}

type bcfComponentXML struct {
	IfcGuid            string `xml:"IfcGuid,attr"`
	OriginatingSystem  string `xml:"OriginatingSystem,attr,omitempty"`
	AuthoringToolId    string `xml:"AuthoringToolId,attr,omitempty"`
}

// ExportBCFZip creates a BCF 2.1 compliant ZIP file from topics.
func ExportBCFZip(topics []Topic) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// Write bcf.version
	versionData, _ := xml.MarshalIndent(bcfVersion{
		VersionID: "2.1",
		XMLNS:     "http://www.buildingsmart-tech.org/bcf/version/2.1",
	}, "", "  ")
	writeZipFile(w, "bcf.version", []byte(xml.Header+string(versionData)))

	for _, topic := range topics {
		prefix := topic.GUID + "/"

		// Build markup
		markup := bcfMarkup{
			XMLNS: "http://www.buildingsmart-tech.org/bcf/markup/2.1",
			Topic: bcfTopicXML{
				GUID:         topic.GUID,
				TopicType:    derefStr(topic.TopicType),
				TopicStatus:  topic.TopicStatus,
				Title:        topic.Title,
				Description:  derefStr(topic.Description),
				Priority:     derefStr(topic.Priority),
				CreationDate: topic.CreatedAt.Format(time.RFC3339),
				Labels:       topic.Labels,
			},
		}

		// Comments
		for _, c := range topic.Comments {
			cGUID := c.ID // Use ID as GUID for now
			markup.Comment = append(markup.Comment, bcfCommentXML{
				GUID:    cGUID,
				Date:    c.CreatedAt.Format(time.RFC3339),
				Author:  derefStr(c.AuthorName),
				Comment: c.Body,
			})
		}

		// Viewpoints
		for i, vp := range topic.Viewpoints {
			var vpFileName, snapFileName string
			if i == 0 {
				vpFileName = "viewpoint.bcfv"
				snapFileName = "snapshot.png"
			} else {
				vpFileName = vp.GUID + ".bcfv"
				snapFileName = vp.GUID + ".png"
			}

			markup.Viewpoints = append(markup.Viewpoints, bcfViewpointRef{
				GUID:      vp.GUID,
				Viewpoint: vpFileName,
				Snapshot:  snapFileName,
			})

			// Write viewpoint .bcfv file
			visInfo := bcfVisInfo{
				XMLNS: "http://www.buildingsmart-tech.org/bcf/viewpoint/2.1",
				GUID:  vp.GUID,
			}

			if vp.CameraType == "perspective" {
				fov := 60.0
				if vp.FieldOfView != nil {
					fov = *vp.FieldOfView
				}
				visInfo.PerspectiveCamera = &bcfPerspective{
					CameraViewPoint: bcfPoint{vp.CameraPosition.X, vp.CameraPosition.Y, vp.CameraPosition.Z},
					CameraDirection: bcfPoint{vp.CameraDirection.X, vp.CameraDirection.Y, vp.CameraDirection.Z},
					CameraUpVector:  bcfPoint{vp.CameraUp.X, vp.CameraUp.Y, vp.CameraUp.Z},
					FieldOfView:     fov,
				}
			} else {
				scale := 1.0
				if vp.ViewWorldScale != nil {
					scale = *vp.ViewWorldScale
				}
				visInfo.OrthogonalCamera = &bcfOrthogonal{
					CameraViewPoint:  bcfPoint{vp.CameraPosition.X, vp.CameraPosition.Y, vp.CameraPosition.Z},
					CameraDirection:  bcfPoint{vp.CameraDirection.X, vp.CameraDirection.Y, vp.CameraDirection.Z},
					CameraUpVector:   bcfPoint{vp.CameraUp.X, vp.CameraUp.Y, vp.CameraUp.Z},
					ViewToWorldScale: scale,
				}
			}

			vpData, _ := xml.MarshalIndent(visInfo, "", "  ")
			writeZipFile(w, prefix+vpFileName, []byte(xml.Header+string(vpData)))

			// Write snapshot if available
			if vp.SnapshotBase64 != nil {
				snapData := decodeBase64DataURL(*vp.SnapshotBase64)
				if len(snapData) > 0 {
					writeZipFile(w, prefix+snapFileName, snapData)
				}
			}
		}

		// Write markup.bcf
		markupData, _ := xml.MarshalIndent(markup, "", "  ")
		writeZipFile(w, prefix+"markup.bcf", []byte(xml.Header+string(markupData)))
	}

	w.Close()
	return buf.Bytes(), nil
}

// ParseBCFZip parses a BCF 2.1 ZIP file and returns topics.
func ParseBCFZip(data []byte) ([]Topic, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	// Index files by path
	files := make(map[string]*zip.File)
	for _, f := range r.File {
		files[f.Name] = f
	}

	var topics []Topic

	// Find all markup.bcf files
	for path, f := range files {
		if !strings.HasSuffix(path, "/markup.bcf") {
			continue
		}

		topicDir := strings.TrimSuffix(path, "markup.bcf")

		rc, err := f.Open()
		if err != nil {
			continue
		}

		var markup bcfMarkup
		if err := xml.NewDecoder(rc).Decode(&markup); err != nil {
			rc.Close()
			continue
		}
		rc.Close()

		topic := Topic{
			GUID:        markup.Topic.GUID,
			Title:       markup.Topic.Title,
			TopicStatus: markup.Topic.TopicStatus,
		}

		if markup.Topic.Description != "" {
			topic.Description = &markup.Topic.Description
		}
		if markup.Topic.Priority != "" {
			topic.Priority = &markup.Topic.Priority
		}
		if markup.Topic.TopicType != "" {
			topic.TopicType = &markup.Topic.TopicType
		}

		// Parse viewpoints
		for _, vpRef := range markup.Viewpoints {
			vpPath := topicDir + vpRef.Viewpoint
			vpFile, ok := files[vpPath]
			if !ok {
				continue
			}

			vpRC, err := vpFile.Open()
			if err != nil {
				continue
			}

			var visInfo bcfVisInfo
			if err := xml.NewDecoder(vpRC).Decode(&visInfo); err != nil {
				vpRC.Close()
				continue
			}
			vpRC.Close()

			vp := Viewpoint{
				GUID: visInfo.GUID,
			}

			if visInfo.PerspectiveCamera != nil {
				cam := visInfo.PerspectiveCamera
				vp.CameraType = "perspective"
				vp.CameraPosition = Vector3{cam.CameraViewPoint.X, cam.CameraViewPoint.Y, cam.CameraViewPoint.Z}
				vp.CameraDirection = Vector3{cam.CameraDirection.X, cam.CameraDirection.Y, cam.CameraDirection.Z}
				vp.CameraUp = Vector3{cam.CameraUpVector.X, cam.CameraUpVector.Y, cam.CameraUpVector.Z}
				vp.FieldOfView = &cam.FieldOfView
			} else if visInfo.OrthogonalCamera != nil {
				cam := visInfo.OrthogonalCamera
				vp.CameraType = "orthogonal"
				vp.CameraPosition = Vector3{cam.CameraViewPoint.X, cam.CameraViewPoint.Y, cam.CameraViewPoint.Z}
				vp.CameraDirection = Vector3{cam.CameraDirection.X, cam.CameraDirection.Y, cam.CameraDirection.Z}
				vp.CameraUp = Vector3{cam.CameraUpVector.X, cam.CameraUpVector.Y, cam.CameraUpVector.Z}
				vp.ViewWorldScale = &cam.ViewToWorldScale
			}

			// Load snapshot
			if vpRef.Snapshot != "" {
				snapPath := topicDir + vpRef.Snapshot
				if snapFile, ok := files[snapPath]; ok {
					if snapRC, err := snapFile.Open(); err == nil {
						snapBuf := new(bytes.Buffer)
						snapBuf.ReadFrom(snapRC)
						snapRC.Close()
						encoded := "data:image/png;base64," + base64.StdEncoding.EncodeToString(snapBuf.Bytes())
						vp.SnapshotBase64 = &encoded
					}
				}
			}

			topic.Viewpoints = append(topic.Viewpoints, vp)
		}

		// Parse comments
		for _, c := range markup.Comment {
			comment := Comment{
				Body: c.Comment,
			}
			if c.Author != "" {
				comment.AuthorName = &c.Author
			}
			topic.Comments = append(topic.Comments, comment)
		}

		topics = append(topics, topic)
	}

	return topics, nil
}

// --- Helpers ---

func writeZipFile(w *zip.Writer, name string, data []byte) {
	f, err := w.Create(name)
	if err != nil {
		return
	}
	f.Write(data)
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func encodeBase64(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func decodeBase64DataURL(dataURL string) []byte {
	// Strip "data:image/png;base64," prefix
	idx := strings.Index(dataURL, ",")
	if idx < 0 {
		return nil
	}
	data, err := base64.StdEncoding.DecodeString(dataURL[idx+1:])
	if err != nil {
		return nil
	}
	return data
}
