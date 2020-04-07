import {
  faFacebookSquare,
  faGithubSquare,
  faRedditSquare,
  faTwitterSquare,
  faYoutubeSquare,
} from "@fortawesome/free-brands-svg-icons"
import { Account } from "../models/account"
import { SocialNetwork } from "../atoms/social-links/social-links.component"

export const SOCIAL_NETWORK_BASE_URL = {
  twitter: "https://www.twitter.com/",
  youtube: "https://youtube.com/c/",
  facebook: "https://www.facebook.com/",
  github: "https://github.com/",
  reddit: "https://www.reddit.com/user/",
  // steemit: "https://steemit.com/@"
}

export const SOCIAL_ICON_MAP = {
  twitter: faTwitterSquare,
  facebook: faFacebookSquare,
  github: faGithubSquare,
  youtube: faYoutubeSquare,
  reddit: faRedditSquare,
}

export const SOCIAL_NETWORKS = ["twitter", "facebook", "github", "youtube", "reddit"]

export function processSocialNetworkNames(account: Account): SocialNetwork[] {
  const socialNetworks: SocialNetwork[] = []
  if (!account.account_verifications) {
    return processDefaultSocialNetworkNames(account)
  }

  SOCIAL_NETWORKS.forEach((socialNetwork: string) => {
    if (
      account.account_verifications![socialNetwork] != null &&
      account.account_verifications![socialNetwork].handle !== ""
    ) {
      socialNetworks.push({
        url: `${SOCIAL_NETWORK_BASE_URL.facebook}${
          account.account_verifications![socialNetwork].handle
        }`,
        name: SOCIAL_ICON_MAP.facebook,
        verified: account.account_verifications![socialNetwork].verified,
      })
    }
  })

  if (socialNetworks.length > 0) {
    return socialNetworks
  }

  return processDefaultSocialNetworkNames(account)
}

export function processDefaultSocialNetworkNames(account: Account) {
  if (!account.block_producer_info) {
    return []
  }

  const networkHandles = account.block_producer_info.org.social
  return Object.keys(networkHandles)
    .filter((key) => !!SOCIAL_NETWORK_BASE_URL[key])
    .map((key) => {
      return {
        url: `${SOCIAL_NETWORK_BASE_URL[key]}${networkHandles[key]}`,
        name: SOCIAL_ICON_MAP[key],
        verified: false,
      }
    })
}
