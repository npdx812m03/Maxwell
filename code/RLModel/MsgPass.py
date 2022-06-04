import tensorflow as tf
import numpy as np

def get_ajd_mat(adj_mat):
    msg_mats = []
    for i, col in enumerate(adj_mat):
        for j, v in enumerate(col):
            if v == 1:
                indices = np.mat([[i], [j]]).transpose()
                sp_mat = tf.SparseTensorValue(indices, 1, (len(adj_mat), len(adj_mat[0])))
                msg_mats.append(sp_mat)
    return msg_mats
