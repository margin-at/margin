export interface UserProfile {
  did: string;
  handle: string;
  displayName?: string;
  description?: string;
  avatar?: string;
  banner?: string;
  website?: string;
  links?: string[];
  followersCount?: number;
  followsCount?: number;
  postsCount?: number;
  labels?: ContentLabel[];
}

interface BaseSelector {
  /** A further selector applied within the region this selector identifies (W3C refinement). */
  refinedBy?: Selector;
}

export interface TextQuoteSelector extends BaseSelector {
  type: "TextQuoteSelector";
  exact: string;
  prefix?: string;
  suffix?: string;
}

export interface TextPositionSelector extends BaseSelector {
  type: "TextPositionSelector";
  start: number;
  end: number;
}

export interface FragmentSelector extends BaseSelector {
  type: "FragmentSelector";
  value: string;
  conformsTo?: string;
}

export interface CssSelector extends BaseSelector {
  type: "CssSelector";
  value: string;
}

export interface XPathSelector extends BaseSelector {
  type: "XPathSelector";
  value: string;
}

export interface RangeSelector extends BaseSelector {
  type: "RangeSelector";
  startSelector: Selector;
  endSelector: Selector;
}

export type Selector =
  | TextQuoteSelector
  | TextPositionSelector
  | FragmentSelector
  | CssSelector
  | XPathSelector
  | RangeSelector;

/** Narrows a selector to a TextQuoteSelector if possible, for reading quote text. */
export function asTextQuote(
  selector?: Selector | null,
): TextQuoteSelector | undefined {
  return selector?.type === "TextQuoteSelector" ? selector : undefined;
}

export interface Target {
  source: string;
  title?: string;
  selector?: Selector;
}

export interface AnnotationBody {
  type: "TextualBody";
  value: string;
  format: "text/plain";
}

export interface Param {
  id: string;
  value: string;
}

export interface AnnotationItem {
  uri: string;
  id?: string;
  cid: string;
  author: UserProfile;
  creator?: UserProfile;
  target?: Target;
  source?: string;
  body?: AnnotationBody;
  motivation: "highlighting" | "commenting" | "bookmarking" | string;
  type?: string;
  createdAt: string;
  text?: string;
  title?: string;
  description?: string;
  color?: string;
  tags?: string[];
  editedAt?: string;
  likeCount?: number;
  replyCount?: number;
  repostCount?: number;
  children?: AnnotationItem[];
  inReplyTo?: string;
  viewer?: {
    like?: string;
  };
  collection?: {
    uri: string;
    name: string;
    icon?: string;
  };
  context?: {
    uri: string;
    name: string;
    icon?: string;
  }[];
  addedBy?: UserProfile;
  collectionItemUri?: string;
  reply?: {
    parent?: {
      uri: string;
      cid: string;
    };
    root?: {
      uri: string;
      cid: string;
    };
  };
  parentUri?: string;
  rootUri?: string;
  labels?: ContentLabel[];
}

export type ActorSearchItem = UserProfile;

export interface FeedResponse {
  items: AnnotationItem[];
  hasMore: boolean;
  fetchedCount: number;
}

export interface NotificationItem {
  id: number;
  recipient: UserProfile;
  actor: UserProfile;
  type:
    | "reply"
    | "quote"
    | "highlight"
    | "bookmark"
    | "annotation"
    | "like"
    | "follow";
  subjectUri: string;
  subject?: AnnotationItem | unknown;
  createdAt: string;
  readAt?: string;
}

export interface Collection {
  id: string;
  uri: string;
  name: string;
  description?: string;
  icon?: string;
  creator: UserProfile;
  createdAt: string;
  itemCount: number;
  items?: AnnotationItem[];
}

export interface CollectionItem {
  id: string;
  collectionId: string;
  subjectUri: string;
  createdAt: string;
  annotation?: AnnotationItem;
}

export interface ModerationRelationship {
  blocking: boolean;
  muting: boolean;
  blockedBy: boolean;
}

export interface BlockedUser {
  did: string;
  author: UserProfile;
  createdAt: string;
}

export interface MutedUser {
  did: string;
  author: UserProfile;
  createdAt: string;
}

export interface ModerationReport {
  id: number;
  reporter: UserProfile;
  subject: UserProfile;
  subjectUri?: string;
  reasonType: string;
  reasonText?: string;
  status: string;
  createdAt: string;
  resolvedAt?: string;
  resolvedBy?: string;
}

export type ReportReasonType =
  | "spam"
  | "violation"
  | "misleading"
  | "sexual"
  | "rude"
  | "other";

export interface ContentLabel {
  val: string;
  src: string;
  scope?: "account" | "content";
}

export type ContentLabelValue =
  | "sexual"
  | "nudity"
  | "violence"
  | "gore"
  | "spam"
  | "misleading";

export type LabelVisibility = "hide" | "warn" | "ignore";

export interface LabelerSubscription {
  did: string;
}

export interface LabelPreference {
  labelerDid: string;
  label: string;
  visibility: LabelVisibility;
}

export interface LabelDefinition {
  identifier: string;
  severity: string;
  blurs: string;
  description: string;
}

export interface LabelerInfo {
  did: string;
  name: string;
  labels: LabelDefinition[];
}

export interface HydratedLabel {
  id: number;
  src: string;
  uri: string;
  val: string;
  createdBy: {
    did: string;
    handle: string;
    displayName?: string;
    avatar?: string;
  };
  createdAt: string;
  subject?: {
    did: string;
    handle: string;
    displayName?: string;
    avatar?: string;
  };
}
export interface EditHistoryItem {
  id: number;
  uri: string;
  recordType: string;
  previousContent: string;
  previousCid?: string;
  editedAt: string;
}
