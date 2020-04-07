import { PillComponentClass } from "./generic-pill.component"
import { BuyRamPillComponent } from "./system/buy-ram-pill.component"
import { BuyRamBytesPillComponent } from "./system/buy-ram-bytes-pill.component"
import { RefundPillComponent } from "./system/refund-pill.component"
import { DelegateBandwidthPillComponent } from "./system/delegate-bandwidth-pill.component"
import { IssuePillComponent } from "./system/issue-pill.component"
import { TransferPillComponent } from "./transfer-pill.component"
import { VotePillComponent } from "./system/vote-pill.component"
import { UnDelegateBandwidthPillComponent } from "./system/undelegate-bandwidth-pill.component"
import { NewAccountPillComponent } from "./system/newaccount-pill.component"
import { LinkAuthPillComponent } from "./system/linkauth-pill.component"
import { UpdateAuthPillComponent } from "./system/updateauth-pill.component"
import { ClaimRewardsPillComponent } from "./system/claim-rewards-pill.component"
import { SetcodePillComponent } from "./system/setcode-pill.component"
import { RegProxyPillComponent } from "./system/regproxy-pill.component"
import { ResolveBetPillComponent } from "./eosbetdice11/resolvebet-pill.component"
import { BetReceiptPillComponent } from "./eosbetdice11/betreceipt-pill.component"
import { ForumProposePillComponent } from "./forum/forum-propose-pill.component"
import { ForumPostPillComponent } from "./forum/forum-post-pill.component"
import { DecenTwitterTweetPillComponent } from "./decenttwitter/decenttwitter-tweet-pill.component"
import { SenseGenesisTransferPillComponent } from "./sense/sense-genesis-transfer-pill.component"
import { DmailTransferPillComponent } from "./d.mail/dmail-transfer-pill.component"
import { PixeosClaimRewardsPillComponent } from "./pixeos/pixeos-claimbal-pill.component"
import { PixeosAddClaimPillComponent } from "./pixeos/pixeos-addclaim-pill.component"
import { KarmaTransferPillComponent } from "./karma/karma-transfer-pill.component"
import { KarmaPowerupPillComponent } from "./karma/karma-powerup-pill.component"
import { KarmaClaimPillComponent } from "./karma/karma-claim-pill.component"
import { KarmaClaimPostPillComponent } from "./karma/karma-claim-post-pill.component"
import { KarmaPowerdownPillComponent } from "./karma/karma-powerdown-pill.component"
import { KarmaRefundPillComponent } from "./karma/karma-refund-pill.component"
import { KarmaSetrewardsPillComponent } from "./karma/karma-setrewards-pill.component"
import { CarbonTransferPillComponent } from "./carbon/carbon-transfer-pill.component"
import { CarbonIssuePillComponent } from "./carbon/carbon-issue-pill.component"
import { CarbonBurnPillComponent } from "./carbon/carbon-burn-pill.component"
import { DfuseEventPillComponent } from "./dfuse-events/dfuse-event-pill.component"
import { MurmurCommentMurmurPillComponent } from "./murmur/murmur-commentmurmur-pill.component"
import { MurmurCommentYellPillComponent } from "./murmur/murmur-commentyell-pill.component"
import { MurmurFollowPillComponent } from "./murmur/murmur-follow-pill.component"
import { MurmurMurmurPillComponent } from "./murmur/murmur-murmur-pill.component"
import { SnoopMurmurPillComponent } from "./murmur/snoop-murmur-pill.component"
import { SnoopYellPillComponent } from "./murmur/snoop-yell-pill.component"
import { MurmurUnfollowPillComponent } from "./murmur/murmur-unfollow-pill.component"
import { MurmurWhisperPillComponent } from "./murmur/murmur-whisper-pill.component"
import { MurmurYellPillComponent } from "./murmur/murmur-yell-pill.component"
import { MurmurTransferPillComponent } from "./murmur/murmur-transfer-pill.component"
import { InfiniverseTransferPillComponent } from "./infiniverse/infiniverse-transfer-pill.component"
import { MakeOfferPillComponent } from "./infiniverse/infiniverse-makeoffer-pill.component"
import { MoveLandPillComponent } from "./infiniverse/infiniverse-moveland-pill.component"
import { InfiniversePersistpolyPillComponent } from "./infiniverse/infiniverse-persistpoly-pill.component"
import { InfiniverseRegisterlandPillComponent } from "./infiniverse/infiniverse-registerland-pill.component"
import { InfiniverseSetlandpricePillComponent } from "./infiniverse/infiniverse-setlandprice-pill.component"
import { InfiniverseUpdatePersistPillComponent } from "./infiniverse/infiniverse-updatepersist-pill.component"
import { InfiniverseDeletePersistPillComponent } from "./infiniverse/infiniverse-deletepersis-pill.component"

export const ALL_TEMPLATES: PillComponentClass[] = [
  BuyRamPillComponent,
  BuyRamBytesPillComponent,
  RefundPillComponent,
  DelegateBandwidthPillComponent,
  IssuePillComponent,
  TransferPillComponent,
  VotePillComponent,
  UnDelegateBandwidthPillComponent,
  NewAccountPillComponent,
  LinkAuthPillComponent,
  UpdateAuthPillComponent,
  ClaimRewardsPillComponent,
  SetcodePillComponent,
  RegProxyPillComponent,
  ResolveBetPillComponent,
  BetReceiptPillComponent,
  ForumProposePillComponent,
  ForumPostPillComponent,
  DecenTwitterTweetPillComponent,

  // Custom components with logos
  SenseGenesisTransferPillComponent,
  DmailTransferPillComponent,
  PixeosClaimRewardsPillComponent,
  PixeosAddClaimPillComponent,
  KarmaTransferPillComponent,
  KarmaPowerupPillComponent,
  KarmaClaimPillComponent,
  KarmaClaimPostPillComponent,
  KarmaPowerdownPillComponent,
  KarmaRefundPillComponent,
  KarmaSetrewardsPillComponent,
  CarbonTransferPillComponent,
  CarbonIssuePillComponent,
  CarbonBurnPillComponent,
  DfuseEventPillComponent,

  MurmurCommentMurmurPillComponent,
  MurmurCommentYellPillComponent,
  MurmurFollowPillComponent,
  MurmurMurmurPillComponent,
  SnoopMurmurPillComponent,
  SnoopYellPillComponent,
  MurmurUnfollowPillComponent,
  MurmurWhisperPillComponent,
  MurmurYellPillComponent,
  MurmurTransferPillComponent,
  InfiniverseTransferPillComponent,
  MakeOfferPillComponent,
  MoveLandPillComponent,
  InfiniversePersistpolyPillComponent,
  InfiniverseRegisterlandPillComponent,
  InfiniverseSetlandpricePillComponent,
  InfiniverseUpdatePersistPillComponent,
  InfiniverseDeletePersistPillComponent
]
