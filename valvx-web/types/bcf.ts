/** BCF 2.1 compliant types for BIM Collaboration Format */

export interface BcfViewpoint {
  id: string
  guid: string
  topicId: string
  cameraType: 'perspective' | 'orthogonal'
  cameraPosition: { x: number; y: number; z: number }
  cameraDirection: { x: number; y: number; z: number }
  cameraUp: { x: number; y: number; z: number }
  fieldOfView?: number
  viewWorldScale?: number
  snapshotBase64?: string
  components?: BcfComponents
  clippingPlanes?: BcfClippingPlane[]
  lines?: BcfLine[]
  createdAt: string
}

export interface BcfComponents {
  selection?: BcfComponent[]
  visibility?: {
    defaultVisibility: boolean
    exceptions: BcfComponent[]
  }
  coloring?: {
    color: string
    components: BcfComponent[]
  }[]
}

export interface BcfComponent {
  ifcGuid: string
  originatingSystem?: string
  authoringToolId?: string
}

export interface BcfClippingPlane {
  location: { x: number; y: number; z: number }
  direction: { x: number; y: number; z: number }
}

export interface BcfLine {
  start: { x: number; y: number; z: number }
  end: { x: number; y: number; z: number }
}

export interface BcfTopic {
  id: string
  guid: string
  title: string
  description?: string
  priority?: 'Critical' | 'Major' | 'Normal' | 'Minor'
  topicType?: 'Issue' | 'Request' | 'Comment' | 'Clash'
  topicStatus?: 'Open' | 'InProgress' | 'Closed' | 'ReOpened'
  stage?: string
  assignedTo?: string
  assignedToName?: string
  dueDate?: string
  labels?: string[]
  projectId: string
  creatorId: string
  creatorName?: string
  modifiedBy?: string
  viewpoints?: BcfViewpoint[]
  comments?: BcfComment[]
  fileVersionIds?: string[]
  createdAt: string
  updatedAt: string
}

export interface BcfComment {
  id: string
  body: string
  viewpointId?: string
  topicId: string
  authorId: string
  authorName?: string
  createdAt: string
  updatedAt: string
}

export interface BcfCreateTopicRequest {
  title: string
  description?: string
  priority?: BcfTopic['priority']
  topicType?: BcfTopic['topicType']
  assignedTo?: string
  dueDate?: string
  labels?: string[]
  fileVersionIds?: string[]
  viewpoint?: BcfCreateViewpointRequest
}

export interface BcfCreateViewpointRequest {
  cameraType: 'perspective' | 'orthogonal'
  cameraPosition: { x: number; y: number; z: number }
  cameraDirection: { x: number; y: number; z: number }
  cameraUp: { x: number; y: number; z: number }
  fieldOfView?: number
  viewWorldScale?: number
  snapshotBase64?: string
  components?: BcfComponents
  clippingPlanes?: BcfClippingPlane[]
}

export interface BcfCreateCommentRequest {
  body: string
  viewpointId?: string
}
