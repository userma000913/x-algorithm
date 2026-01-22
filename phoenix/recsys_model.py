# Copyright 2026 X.AI Corp.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import logging
from dataclasses import dataclass
from typing import Any, NamedTuple, Optional, Tuple

import haiku as hk
import jax
import jax.numpy as jnp

from grok import (
    TransformerConfig,
    Transformer,
    layer_norm,
)

logger = logging.getLogger(__name__)


@dataclass
class HashConfig:
    """基于哈希的嵌入配置。"""

    num_user_hashes: int = 2
    num_item_hashes: int = 2
    num_author_hashes: int = 2


@dataclass
class RecsysEmbeddings:
    """来自嵌入表的预查找嵌入容器。

    这些嵌入在传递给模型之前从哈希表中查找。
    block_*_reduce 函数将多个哈希嵌入组合成单个表示。
    """

    user_embeddings: jax.typing.ArrayLike
    history_post_embeddings: jax.typing.ArrayLike
    candidate_post_embeddings: jax.typing.ArrayLike
    history_author_embeddings: jax.typing.ArrayLike
    candidate_author_embeddings: jax.typing.ArrayLike


class RecsysModelOutput(NamedTuple):
    """推荐模型的输出。"""

    logits: jax.Array


class RecsysBatch(NamedTuple):
    """推荐模型的输入批次。

    包含特征数据（哈希、动作、产品表面），但不包含嵌入。
    嵌入通过 RecsysEmbeddings 单独传递。
    """

    user_hashes: jax.typing.ArrayLike
    history_post_hashes: jax.typing.ArrayLike
    history_author_hashes: jax.typing.ArrayLike
    history_actions: jax.typing.ArrayLike
    history_product_surface: jax.typing.ArrayLike
    candidate_post_hashes: jax.typing.ArrayLike
    candidate_author_hashes: jax.typing.ArrayLike
    candidate_product_surface: jax.typing.ArrayLike


def block_user_reduce(
    user_hashes: jnp.ndarray,
    user_embeddings: jnp.ndarray,
    num_user_hashes: int,
    emb_size: int,
    embed_init_scale: float = 1.0,
) -> Tuple[jax.Array, jax.Array]:
    """将多个用户哈希嵌入组合成单个用户表示。

    参数:
        user_hashes: [B, num_user_hashes] - 哈希值（0 = 无效/填充）
        user_embeddings: [B, num_user_hashes, D] - 查找的嵌入
        num_user_hashes: 使用的哈希函数数量
        emb_size: 嵌入维度 D
        embed_init_scale: 投影的初始化缩放

    返回:
        user_embedding: [B, 1, D] - 组合的用户嵌入
        user_padding_mask: [B, 1] - 用户有效时为 True
    """
    B = user_embeddings.shape[0]
    D = emb_size

    user_embedding = user_embeddings.reshape((B, 1, num_user_hashes * D))

    embed_init = hk.initializers.VarianceScaling(embed_init_scale, mode="fan_out")
    proj_mat_1 = hk.get_parameter(
        "proj_mat_1",
        [num_user_hashes * D, D],
        dtype=jnp.float32,
        init=lambda shape, dtype: embed_init(list(reversed(shape)), dtype).T,
    )

    user_embedding = jnp.dot(user_embedding.astype(proj_mat_1.dtype), proj_mat_1).astype(
        user_embeddings.dtype
    )

    # 哈希 0 保留用于填充
    user_padding_mask = (user_hashes[:, 0] != 0).reshape(B, 1).astype(jnp.bool_)

    return user_embedding, user_padding_mask


def block_history_reduce(
    history_post_hashes: jnp.ndarray,
    history_post_embeddings: jnp.ndarray,
    history_author_embeddings: jnp.ndarray,
    history_product_surface_embeddings: jnp.ndarray,
    history_actions_embeddings: jnp.ndarray,
    num_item_hashes: int,
    num_author_hashes: int,
    embed_init_scale: float = 1.0,
) -> Tuple[jax.Array, jax.Array]:
    """将历史嵌入（帖子、作者、动作、产品表面）组合成序列。

    参数:
        history_post_hashes: [B, S, num_item_hashes]
        history_post_embeddings: [B, S, num_item_hashes, D]
        history_author_embeddings: [B, S, num_author_hashes, D]
        history_product_surface_embeddings: [B, S, D]
        history_actions_embeddings: [B, S, D]
        num_item_hashes: 项目的哈希函数数量
        num_author_hashes: 作者的哈希函数数量
        emb_size: 嵌入维度 D
        embed_init_scale: 初始化缩放

    返回:
        history_embeddings: [B, S, D]
        history_padding_mask: [B, S]
    """
    B, S, _, D = history_post_embeddings.shape

    history_post_embeddings_reshaped = history_post_embeddings.reshape((B, S, num_item_hashes * D))
    history_author_embeddings_reshaped = history_author_embeddings.reshape(
        (B, S, num_author_hashes * D)
    )

    post_author_embedding = jnp.concatenate(
        [
            history_post_embeddings_reshaped,
            history_author_embeddings_reshaped,
            history_actions_embeddings,
            history_product_surface_embeddings,
        ],
        axis=-1,
    )

    embed_init = hk.initializers.VarianceScaling(embed_init_scale, mode="fan_out")
    proj_mat_3 = hk.get_parameter(
        "proj_mat_3",
        [post_author_embedding.shape[-1], D],
        dtype=jnp.float32,
        init=lambda shape, dtype: embed_init(list(reversed(shape)), dtype).T,
    )

    history_embedding = jnp.dot(post_author_embedding.astype(proj_mat_3.dtype), proj_mat_3).astype(
        post_author_embedding.dtype
    )

    history_embedding = history_embedding.reshape(B, S, D)

    history_padding_mask = (history_post_hashes[:, :, 0] != 0).reshape(B, S)

    return history_embedding, history_padding_mask


def block_candidate_reduce(
    candidate_post_hashes: jnp.ndarray,
    candidate_post_embeddings: jnp.ndarray,
    candidate_author_embeddings: jnp.ndarray,
    candidate_product_surface_embeddings: jnp.ndarray,
    num_item_hashes: int,
    num_author_hashes: int,
    embed_init_scale: float = 1.0,
) -> Tuple[jax.Array, jax.Array]:
    """将候选嵌入（帖子、作者、产品表面）组合成序列。

    参数:
        candidate_post_hashes: [B, C, num_item_hashes]
        candidate_post_embeddings: [B, C, num_item_hashes, D]
        candidate_author_embeddings: [B, C, num_author_hashes, D]
        candidate_product_surface_embeddings: [B, C, D]
        num_item_hashes: 项目的哈希函数数量
        num_author_hashes: 作者的哈希函数数量
        emb_size: 嵌入维度 D
        embed_init_scale: 初始化缩放

    返回:
        candidate_embeddings: [B, C, D]
        candidate_padding_mask: [B, C]
    """
    B, C, _, D = candidate_post_embeddings.shape

    candidate_post_embeddings_reshaped = candidate_post_embeddings.reshape(
        (B, C, num_item_hashes * D)
    )
    candidate_author_embeddings_reshaped = candidate_author_embeddings.reshape(
        (B, C, num_author_hashes * D)
    )

    post_author_embedding = jnp.concatenate(
        [
            candidate_post_embeddings_reshaped,
            candidate_author_embeddings_reshaped,
            candidate_product_surface_embeddings,
        ],
        axis=-1,
    )

    embed_init = hk.initializers.VarianceScaling(embed_init_scale, mode="fan_out")
    proj_mat_2 = hk.get_parameter(
        "proj_mat_2",
        [post_author_embedding.shape[-1], D],
        dtype=jnp.float32,
        init=lambda shape, dtype: embed_init(list(reversed(shape)), dtype).T,
    )

    candidate_embedding = jnp.dot(
        post_author_embedding.astype(proj_mat_2.dtype), proj_mat_2
    ).astype(post_author_embedding.dtype)

    candidate_padding_mask = (candidate_post_hashes[:, :, 0] != 0).reshape(B, C).astype(jnp.bool_)

    return candidate_embedding, candidate_padding_mask


@dataclass
class PhoenixModelConfig:
    """推荐系统模型的配置。"""

    model: TransformerConfig
    emb_size: int
    num_actions: int
    history_seq_len: int = 128
    candidate_seq_len: int = 32

    name: Optional[str] = None
    fprop_dtype: Any = jnp.bfloat16

    hash_config: HashConfig = None  # type: ignore

    product_surface_vocab_size: int = 16

    _initialized = False

    def __post_init__(self):
        if self.hash_config is None:
            self.hash_config = HashConfig()

    def initialize(self):
        self._initialized = True
        return self

    def make(self):
        if not self._initialized:
            logger.warning(f"PhoenixModel {self.name} is not initialized. Initializing.")
            self.initialize()

        return PhoenixModel(
            model=self.model.make(),
            config=self,
            fprop_dtype=self.fprop_dtype,
        )


@dataclass
class PhoenixModel(hk.Module):
    """基于 Transformer 的推荐模型，用于对候选进行排序。"""

    model: Transformer
    config: PhoenixModelConfig
    fprop_dtype: Any = jnp.bfloat16
    name: Optional[str] = None

    def _get_action_embeddings(
        self,
        actions: jax.Array,
    ) -> jax.Array:
        """将多热动作向量转换为嵌入。

        使用学习的投影矩阵将有符号动作向量映射到嵌入维度。
        这适用于任意数量的动作。
        """
        config = self.config
        _, _, num_actions = actions.shape
        D = config.emb_size

        embed_init = hk.initializers.VarianceScaling(1.0, mode="fan_out")
        action_projection = hk.get_parameter(
            "action_projection",
            [num_actions, D],
            dtype=jnp.float32,
            init=embed_init,
        )

        actions_signed = (2 * actions - 1).astype(jnp.float32)

        action_emb = jnp.dot(actions_signed.astype(action_projection.dtype), action_projection)

        valid_mask = jnp.any(actions, axis=-1, keepdims=True)
        action_emb = action_emb * valid_mask

        return action_emb.astype(self.fprop_dtype)

    def _single_hot_to_embeddings(
        self,
        input: jax.Array,
        vocab_size: int,
        emb_size: int,
        name: str,
    ) -> jax.Array:
        """通过查找表将单热索引转换为嵌入。

        参数:
            input: [B, S] 分类索引张量
            vocab_size: 词汇表大小
            emb_size: 嵌入维度
            name: 嵌入表参数的名称

        返回:
            embeddings: [B, S, emb_size]
        """
        embed_init = hk.initializers.VarianceScaling(1.0, mode="fan_out")
        embedding_table = hk.get_parameter(
            name,
            [vocab_size, emb_size],
            dtype=jnp.float32,
            init=embed_init,
        )

        input_one_hot = jax.nn.one_hot(input, vocab_size)
        output = jnp.dot(input_one_hot, embedding_table)
        return output.astype(self.fprop_dtype)

    def _get_unembedding(self) -> jax.Array:
        """获取用于解码为 logits 的反嵌入矩阵。"""
        config = self.config
        embed_init = hk.initializers.VarianceScaling(1.0, mode="fan_out")
        unembed_mat = hk.get_parameter(
            "unembeddings",
            [config.emb_size, config.num_actions],
            dtype=jnp.float32,
            init=embed_init,
        )
        return unembed_mat

    def build_inputs(
        self,
        batch: RecsysBatch,
        recsys_embeddings: RecsysEmbeddings,
    ) -> Tuple[jax.Array, jax.Array, int]:
        """从批次和预查找嵌入构建输入嵌入。

        参数:
            batch: 包含哈希、动作、产品表面的 RecsysBatch
            recsys_embeddings: 包含预查找嵌入的 RecsysEmbeddings

        返回:
            embeddings: [B, 1 + history_len + num_candidates, D]
            padding_mask: [B, 1 + history_len + num_candidates]
            candidate_start_offset: int - 候选开始的位置
        """
        config = self.config
        hash_config = config.hash_config

        history_product_surface_embeddings = self._single_hot_to_embeddings(
            batch.history_product_surface,  # type: ignore
            config.product_surface_vocab_size,
            config.emb_size,
            "product_surface_embedding_table",
        )
        candidate_product_surface_embeddings = self._single_hot_to_embeddings(
            batch.candidate_product_surface,  # type: ignore
            config.product_surface_vocab_size,
            config.emb_size,
            "product_surface_embedding_table",
        )

        history_actions_embeddings = self._get_action_embeddings(batch.history_actions)  # type: ignore

        user_embeddings, user_padding_mask = block_user_reduce(
            batch.user_hashes,  # type: ignore
            recsys_embeddings.user_embeddings,  # type: ignore
            hash_config.num_user_hashes,
            config.emb_size,
            1.0,
        )

        history_embeddings, history_padding_mask = block_history_reduce(
            batch.history_post_hashes,  # type: ignore
            recsys_embeddings.history_post_embeddings,  # type: ignore
            recsys_embeddings.history_author_embeddings,  # type: ignore
            history_product_surface_embeddings,
            history_actions_embeddings,
            hash_config.num_item_hashes,
            hash_config.num_author_hashes,
            1.0,
        )

        candidate_embeddings, candidate_padding_mask = block_candidate_reduce(
            batch.candidate_post_hashes,  # type: ignore
            recsys_embeddings.candidate_post_embeddings,  # type: ignore
            recsys_embeddings.candidate_author_embeddings,  # type: ignore
            candidate_product_surface_embeddings,
            hash_config.num_item_hashes,
            hash_config.num_author_hashes,
            1.0,
        )

        embeddings = jnp.concatenate(
            [user_embeddings, history_embeddings, candidate_embeddings], axis=1
        )
        padding_mask = jnp.concatenate(
            [user_padding_mask, history_padding_mask, candidate_padding_mask], axis=1
        )

        candidate_start_offset = user_padding_mask.shape[1] + history_padding_mask.shape[1]

        return embeddings.astype(self.fprop_dtype), padding_mask, candidate_start_offset

    def __call__(
        self,
        batch: RecsysBatch,
        recsys_embeddings: RecsysEmbeddings,
    ) -> RecsysModelOutput:
        """对候选进行排序的前向传播。

        参数:
            batch: 包含哈希、动作、产品表面的 RecsysBatch
            recsys_embeddings: 包含预查找嵌入的 RecsysEmbeddings

        返回:
            包含每个候选 logits 的 RecsysModelOutput。形状 = [B, num_candidates, num_actions]
        """
        embeddings, padding_mask, candidate_start_offset = self.build_inputs(
            batch, recsys_embeddings
        )

        # transformer
        model_output = self.model(
            embeddings,
            padding_mask,
            candidate_start_offset=candidate_start_offset,
        )

        out_embeddings = model_output.embeddings

        out_embeddings = layer_norm(out_embeddings)

        candidate_embeddings = out_embeddings[:, candidate_start_offset:, :]

        unembeddings = self._get_unembedding()
        logits = jnp.dot(candidate_embeddings.astype(unembeddings.dtype), unembeddings)
        logits = logits.astype(self.fprop_dtype)

        return RecsysModelOutput(logits=logits)
