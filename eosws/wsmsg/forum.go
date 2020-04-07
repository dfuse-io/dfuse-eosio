// Copyright 2020 dfuse Platform Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package wsmsg

import (
	"github.com/eoscanada/eos-go/forum"
	"github.com/dfuse-io/dfuse-eosio/eosws/mdl"
)

func init() {
	RegisterOutgoingMessage("forum_post", ForumPost{})
	RegisterOutgoingMessage("forum_vote", ForumVote{})
	RegisterOutgoingMessage("forum_proposition", ForumProposition{})
	RegisterIncomingMessage("get_forum_proposition", GetForumProposition{})
}

/// Forum Get Proposition

type GetForumProposition struct {
	CommonIn

	Data struct {
		Title string `json:"title"`
	} `json:"data"`
}

/// Forum Post

type ForumPost struct {
	CommonOut
	Data *forum.Post `json:"data"`
}

func NewForumPost(post *forum.Post) *ForumPost {
	return &ForumPost{
		Data: post,
	}
}

/// Forum Vote

type ForumProposition struct {
	CommonOut
	Data *mdl.Proposition `json:"data"`
}

func NewProposition(proposition *mdl.Proposition) *ForumProposition {
	return &ForumProposition{
		Data: proposition,
	}
}

/// Forum Vote

type ForumVote struct {
	CommonOut
	Data *forum.Vote `json:"data"`
}

func NewForumVote(vote *forum.Vote) *ForumVote {
	return &ForumVote{
		Data: vote,
	}
}
